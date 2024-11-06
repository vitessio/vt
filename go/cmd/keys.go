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
	"github.com/spf13/cobra"
	"github.com/vitessio/vt/go/data"

	"github.com/vitessio/vt/go/keys"
)

func keysCmd() *cobra.Command {
	var inputType string

	cmd := &cobra.Command{
		Use:     "keys file.test",
		Short:   "Runs vexplain keys on all queries of the test file",
		Example: "vt keys file.test",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg := keys.Config{
				FileName: args[0],
			}

			// Set the input type in config based on the flag
			switch inputType {
			case "sql":
				cfg.Loader = data.SQLScriptLoader{}
			case "mysql-log":
				cfg.Loader = data.MySQLLogLoader{}
			default:
				return errors.New("invalid input type: must be 'sql' or 'mysql-log'")
			}

			return keys.Run(cfg)
		},
	}

	cmd.Flags().StringVar(&inputType, "input-type", "sql", "Specifies the type of input file: 'sql' or 'mysql-log'")

	return cmd
}
