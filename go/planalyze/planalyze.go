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

	"vitess.io/vitess/go/test/vschemawrapper"
	"vitess.io/vitess/go/vt/vtenv"
	"vitess.io/vitess/go/vt/vtgate/engine"
	"vitess.io/vitess/go/vt/vtgate/planbuilder"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
)

type (
	Config struct {
		VSchemaFile          string
		VtExplainVschemaFile string
	}

	// Planalyze is the main struct for the planalyze tool.
	// It is a size four slice as we have four different types of analyzed queries
	Planalyze struct {
		Queries [4][]AnalyzedQuery
	}

	Output struct {
		FileType     string          `json:"fileType"`
		PassThrough  []AnalyzedQuery `json:"passThrough"`
		SimpleRouted []AnalyzedQuery `json:"simpleRouted"`
		Complex      []AnalyzedQuery `json:"complex"`
		Unplannable  []AnalyzedQuery `json:"unplannable"`
	}

	AnalyzedQuery struct {
		QueryStructure string
		Complexity     PlanComplexity
		PlanOutput     json.RawMessage
	}

	PlanComplexity int
)

const (
	PassThrough PlanComplexity = iota
	SimpleRouted
	Complex
	Unplannable
)

func (p PlanComplexity) String() string {
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
	return run(os.Stdout, cfg, logFile)
}

func run(out io.Writer, cfg Config, logFile string) error {
	a := cfg.VSchemaFile != ""
	b := cfg.VtExplainVschemaFile != ""
	if a == b {
		return errors.New("specify exactly one of the following flags: -vschema or -vtexplain-vschema")
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

	planalyzer := &Planalyze{
		Queries: [4][]AnalyzedQuery{
			{},
			{},
			{},
			{},
		},
	}

	for _, query := range ko.Queries {
		var plan *engine.Plan
		plan, err = planbuilder.TestBuilder(query.QueryStructure, vw, "")

		res := getPlanRes(err, plan)
		switch {
		case res == Unplannable:
			errBytes, jsonErr := json.Marshal(err.Error())
			if jsonErr != nil {
				return jsonErr
			}
			planalyzer.Queries[res] = append(planalyzer.Queries[res], AnalyzedQuery{
				QueryStructure: query.QueryStructure,
				Complexity:     res,
				PlanOutput:     errBytes,
			})
		case plan.Instructions != nil:
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
				Complexity:     res,
				PlanOutput:     json.RawMessage(b.String()),
			})
		default:
			// if we don't have an instruction, this query is not interesting for planalyze
		}
	}

	type jsonOutput struct {
		FileType     string          `json:"fileType"`
		PassThrough  []AnalyzedQuery `json:"passThrough"`
		SimpleRouted []AnalyzedQuery `json:"simpleRouted"`
		Complex      []AnalyzedQuery `json:"complex"`
		Unplannable  []AnalyzedQuery `json:"unplannable"`
	}
	res := jsonOutput{
		FileType:     "planalyze",
		PassThrough:  planalyzer.Queries[PassThrough],
		SimpleRouted: planalyzer.Queries[SimpleRouted],
		Complex:      planalyzer.Queries[Complex],
		Unplannable:  planalyzer.Queries[Unplannable],
	}

	jsonData, err := json.MarshalIndent(res, "  ", "  ")
	if err != nil {
		return err
	}
	_, err = out.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}

func getPlanRes(err error, plan *engine.Plan) PlanComplexity {
	if err != nil {
		return Unplannable
	}

	var rp *engine.RoutingParameters

	switch prim := plan.Instructions.(type) {
	case *engine.Route:
		rp = prim.RoutingParameters
	case *engine.Update:
		rp = prim.RoutingParameters
	case *engine.Delete:
		rp = prim.RoutingParameters
	case *engine.Insert:
		if prim.InsertCommon.Opcode == engine.InsertUnsharded {
			return PassThrough
		}
		return SimpleRouted
	default:
		return Complex
	}

	if rp.Opcode.IsSingleShard() {
		return PassThrough
	}

	return SimpleRouted
}

func ReadPlanalyzeFile(filename string) (p Output, err error) {
	c, err := os.ReadFile(filename)
	if err != nil {
		return p, fmt.Errorf("error opening file: %w", err)
	}

	err = json.Unmarshal(c, &p)
	if err != nil {
		return p, fmt.Errorf("error parsing json: %w", err)
	}
	return p, nil
}
