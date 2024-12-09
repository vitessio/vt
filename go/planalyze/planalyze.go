/*
Copyright 2024 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package planalyze

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"vitess.io/vitess/go/test/vschemawrapper"
	"vitess.io/vitess/go/vt/vtenv"
	"vitess.io/vitess/go/vt/vtgate/engine"
	"vitess.io/vitess/go/vt/vtgate/planbuilder"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/markdown"
)

type (
	Config struct {
		VSchemaFile          string
		VtExplainVschemaFile string
	}

	// Planalyze is the main struct for the planalyze tool
	Planalyze struct {
		Queries [4][]AnalyzedQuery
	}

	AnalyzedQuery struct {
		QueryStructure string
		FirstQueryRow  int
		Result         planResult
		PlanOutput     string
	}

	planResult int
)

const (
	PassThrough planResult = iota
	SimpleRouted
	Complex
	Unplannable
)

func (p planResult) String() string {
	switch p {
	case PassThrough:
		return "Pass-through"
	case SimpleRouted:
		return "Simple routed"
	case Complex:
		return "Complex routed"
	case Unplannable:
		return "Unplannable"
	}
	return "Unknown"
}

func Run(cfg Config, logFile string) error {
	return run(os.Stdout, cfg, logFile, time.Now())
}

func run(out io.Writer, cfg Config, logFile string, now time.Time) error {
	a := cfg.VSchemaFile != ""
	b := cfg.VtExplainVschemaFile != ""
	if a == b {
		return errors.New("specify exactly one of the following flags: -vschema, -vtexplain-vschema, -sharded")
	}

	_, vschema, err := data.GetKeyspaces(cfg.VSchemaFile, cfg.VtExplainVschemaFile, "main", false)
	if err != nil {
		return err
	}

	ko, err := keys.ReadKeysFile(logFile)
	if err != nil {
		return err
	}

	vw, err := vschemawrapper.NewVschemaWrapper(vtenv.NewTestEnv(), vschema, nil)
	if err != nil {
		return err
	}

	planalyzer := &Planalyze{}

	for _, query := range ko.Queries {
		var plan *engine.Plan
		plan, err = planbuilder.TestBuilder(query.QueryStructure, vw, "")

		res := getPlanRes(err, plan)
		description := engine.PrimitiveToPlanDescription(plan.Instructions, nil)
		b := new(bytes.Buffer)
		enc := json.NewEncoder(b)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		err = enc.Encode(description)
		if err != nil {
			return err
		}
		planalyzer.Queries[res] = append(planalyzer.Queries[res], AnalyzedQuery{
			QueryStructure: query.QueryStructure,
			FirstQueryRow:  query.LineNumbers[0],
			Result:         res,
			PlanOutput:     b.String(),
		})
	}

	return planalyzer.printMarkdown(out, now, logFile)
}

func (planalyzer *Planalyze) printMarkdown(out io.Writer, now time.Time, logFile string) error {
	md := &markdown.MarkDown{}
	msg := `# Query Planning Report

**Date of Analysis**: %s  
**Analyzed File**: ` + "%s" + `

`
	md.Printf(msg, now.Format(time.DateTime), logFile)
	headers := []string{"Plan Complexity", "Count"}
	var rows [][]string
	total := 0
	for _, i := range []planResult{PassThrough, SimpleRouted, Complex, Unplannable} {
		count := len(planalyzer.Queries[i])
		rows = append(rows, []string{i.String(), strconv.Itoa(count)})
		total += count
	}
	rows = append(rows, []string{"Total", strconv.Itoa(total)})
	md.PrintTable(headers, rows)
	md.NewLine()
	for _, typ := range []planResult{SimpleRouted, Complex} {
		for i, query := range planalyzer.Queries[typ] {
			if i == 0 {
				md.Printf("# %s Queries\n\n", typ.String())
			}
			md.Printf("## Query\n\n```sql\n%s\n```\n\n", query.QueryStructure)
			md.Printf("## Plan\n\n```json\n%s\n```\n\n", query.PlanOutput)
			md.NewLine()
		}
	}

	_, err := md.WriteTo(out)
	if err != nil {
		return fmt.Errorf("error writing markdown: %w", err)
	}
	return nil
}

func getPlanRes(err error, plan *engine.Plan) planResult {
	if err != nil {
		return Unplannable
	}

	rb, ok := plan.Instructions.(*engine.Route)
	switch {
	case !ok:
		return Complex
	case rb.Opcode.IsSingleShard():
		return PassThrough
	}

	return SimpleRouted
}
