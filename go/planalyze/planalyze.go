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
	"io"
	"os"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
	"vitess.io/vitess/go/test/vschemawrapper"
	"vitess.io/vitess/go/vt/vtenv"
	"vitess.io/vitess/go/vt/vtgate/planbuilder"
)

type Config struct {
	logFile     string
	vcshemaFile string
}

func Run(cfg Config) error {
	return run(os.Stdout, cfg)
}

func run(out io.Writer, cfg Config) error {
	ko, err := keys.ReadKeysFile(cfg.logFile)
	if err != nil {
		return err
	}

	_, vschema, err := data.ReadVschema(cfg.vcshemaFile, false)
	if err != nil {
		return err
	}
	vw, err := vschemawrapper.NewVschemaWrapper(vtenv.NewTestEnv(), vschema, nil)
	if err != nil {
		return err
	}
	for _, query := range ko.Queries {
		_, err = planbuilder.TestBuilder(query.QueryStructure, vw, "")
		if err != nil {
			return err
		}
	}

	return nil
}
