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

type Config struct {
	VSchemaFile          string
	VtExplainVschemaFile string
}

func Run(cfg Config, logFile string) error {
	return run(os.Stdout, cfg, logFile)
}

func run(out io.Writer, cfg Config, logFile string) error {
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
	for _, query := range ko.Queries {
		var plan *engine.Plan
		plan, err = planbuilder.TestBuilder(query.QueryStructure, vw, "")

		res := getPlanRes(err, plan)
		_, _ = fmt.Fprintf(out, "%s Query: %s\n", res, query.QueryStructure)
	}

	return nil
}

func getPlanRes(err error, plan *engine.Plan) string {
	var res string
	if err != nil {
		res = "FAIL"
	} else {
		rb, ok := plan.Instructions.(*engine.Route)
		switch {
		case !ok:
			res = "VALID"
		case rb.Opcode.IsSingleShard():
			res = "PERFECT"
		default:
			res = "GOOD"
		}
	}
	return res
}
