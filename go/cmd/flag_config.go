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
)

type csvFlags struct {
	header                                                                                                         bool
	queryField, connectionIDField, queryTimeField, lockTimeField, rowsSentField, rowsExaminedField, timestampField int
}

const allowedInputTypes = "'sql', 'mysql-log', 'vtgate-log', 'csv'"

func addInputTypeFlag(cmd *cobra.Command, s *string) {
	*s = "sql"
	cmd.Flags().StringVar(s, "input-type", "sql", fmt.Sprintf("Specifies the type of input file: %s", allowedInputTypes))
}

func addCSVConfigFlag(cmd *cobra.Command, c *csvFlags) {
	cmd.Flags().BoolVar(&c.header, "csv-header", false, "Indicates that the CSV file has a header row")
	cmd.Flags().IntVar(&c.queryField, "csv-query-field", 0, "Column index or name for the query field (required)")
	cmd.Flags().IntVar(&c.connectionIDField, "csv-connection-id-field", 0, "Column index or name for the connection ID field")
	cmd.Flags().IntVar(&c.queryTimeField, "csv-query-time-field", 0, "Column index or name for the query time field")
	cmd.Flags().IntVar(&c.lockTimeField, "csv-lock-time-field", 0, "Column index or name for the lock time field")
	cmd.Flags().IntVar(&c.rowsSentField, "csv-rows-sent-field", 0, "Column index or name for the rows sent field")
	cmd.Flags().IntVar(&c.rowsExaminedField, "csv-rows-examined-field", 0, "Column index or name for the rows examined field")
	cmd.Flags().IntVar(&c.timestampField, "csv-timestamp-field", 0, "Column index or name for the timestamp field")
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

func csvFlagsToConfig(cmd *cobra.Command, flags csvFlags) data.CSVConfig {
	var c data.CSVConfig
	if cmd.Flags().Changed("csv-query-field") {
		c.QueryField = flags.queryField
	}
	if cmd.Flags().Changed("csv-connection-id-field") {
		c.ConnectionIDField = &flags.connectionIDField
	}
	if cmd.Flags().Changed("csv-query-time-field") {
		c.QueryTimeField = &flags.queryTimeField
	}
	if cmd.Flags().Changed("csv-lock-time-field") {
		c.LockTimeField = &flags.lockTimeField
	}
	if cmd.Flags().Changed("csv-rows-sent-field") {
		c.RowsSentField = &flags.rowsSentField
	}
	if cmd.Flags().Changed("csv-rows-examined-field") {
		c.RowsExaminedField = &flags.rowsExaminedField
	}
	if cmd.Flags().Changed("csv-timestamp-field") {
		c.TimestampField = &flags.timestampField
	}
	if cmd.Flags().Changed("csv-header") {
		c.Header = flags.header
	}
	return c
}
