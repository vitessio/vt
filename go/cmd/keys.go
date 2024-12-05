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
	var csvConfig data.CSVConfig

	cmd := &cobra.Command{
		Use:     "keys ",
		Short:   "Runs vexplain keys on all queries of the test file",
		Example: "vt keys file.test",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg := keys.Config{
				FileName: args[0],
			}

			loader, err := configureLoader(inputType, false, csvConfig)
			if err != nil {
				return err
			}
			cfg.Loader = loader

			return keys.Run(cfg)
		},
	}

	addInputTypeFlag(cmd, &inputType)
	addCSVConfigFlag(cmd, &csvConfig)

	return cmd
}

const allowedInputTypes = "'sql', 'mysql-log', 'vtgate-log', 'csv'"

func addInputTypeFlag(cmd *cobra.Command, s *string) {
	*s = "sql"
	cmd.Flags().StringVar(s, "input-type", "sql", fmt.Sprintf("Specifies the type of input file: %s", allowedInputTypes))
}

func addCSVConfigFlag(cmd *cobra.Command, c *data.CSVConfig) {
	cmd.Flags().BoolVar(&c.Header, "csv-header", false, "Indicates that the CSV file has a header row")
	cmd.Flags().IntVar(&c.QueryField, "csv-query-field", -1, "Column index or name for the query field (required)")
	cmd.Flags().IntVar(&c.ConnectionIDField, "csv-connection-id-field", -1, "Column index or name for the connection ID field")
	cmd.Flags().IntVar(&c.QueryTimeField, "csv-query-time-field", -1, "Column index or name for the query time field")
	cmd.Flags().IntVar(&c.LockTimeField, "csv-lock-time-field", -1, "Column index or name for the lock time field")
	cmd.Flags().IntVar(&c.RowsSentField, "csv-rows-sent-field", -1, "Column index or name for the rows sent field")
	cmd.Flags().IntVar(&c.RowsExaminedField, "csv-rows-examined-field", -1, "Column index or name for the rows examined field")
	cmd.Flags().IntVar(&c.TimestampField, "csv-timestamp-field", -1, "Column index or name for the timestamp field")
}

func configureLoader(inputType string, needsBindVars bool, csvConfig data.CSVConfig) (data.Loader, error) {
	switch inputType {
	case "sql":
		return data.SlowQueryLogLoader{}, nil
	case "mysql-log":
		return data.MySQLLogLoader{}, nil
	case "vtgate-log":
		return data.VtGateLogLoader{NeedsBindVars: needsBindVars}, nil
	case "csv":
		if csvConfig.QueryField == -1 {
			return nil, errors.New("must specify query field for CSV loader")
		}
		return data.CSVLogLoader{Config: csvConfig}, nil
	default:
		return nil, fmt.Errorf("invalid input type: must be %s", allowedInputTypes)
	}
}
