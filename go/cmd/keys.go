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

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
)

func keysCmd() *cobra.Command {
	var inputType string

	cmd := &cobra.Command{
		Use:     "keys ",
		Short:   "Runs vexplain keys on all queries of the test file",
		Example: "vt keys file.test",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg := keys.Config{
				FileName: args[0],
			}

			loader, err := configureLoader(inputType, false)
			if err != nil {
				return err
			}
			cfg.Loader = loader

			return keys.Run(cfg)
		},
	}

	addInputTypeFlag(cmd, &inputType)

	return cmd
}

const allowedInputTypes = "'sql', 'mysql-log' or 'vtgate-log'"

func addInputTypeFlag(cmd *cobra.Command, s *string) {
	*s = "sql"
	cmd.Flags().StringVar(s, "input-type", "sql", fmt.Sprintf("Specifies the type of input file: %s", allowedInputTypes))
}

func configureLoader(inputType string, needsBindVars bool) (data.Loader, error) {
	switch inputType {
	case "sql":
		return data.SQLScriptLoader{}, nil
	case "mysql-log":
		return data.MySQLLogLoader{}, nil
	case "vtgate-log":
		return data.VtGateLogLoader{NeedsBindVars: needsBindVars}, nil
	default:
		return nil, errors.New(fmt.Sprintf("invalid input type: must be %s", allowedInputTypes))
	}
}
