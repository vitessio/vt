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
	vttester "github.com/vitessio/vt/go/tester"
)

func testerCmd() *cobra.Command {
	var cfg vttester.Config
	var inputType string
	csvConfig := data.NewEmptyCSVConfig(false, -1)

	cmd := &cobra.Command{
		Aliases: []string{"test"},
		Use:     "tester ",
		Short:   "Test the given workload against both Vitess and MySQL.",
		Example: "vt tester ",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Tests = args
			cfg.Compare = true
			loader, err := configureLoader(inputType, true, csvConfig)
			if err != nil {
				return err
			}
			cfg.Loader = loader

			return usageErr(cmd, vttester.Run(cfg))
		},
	}

	commonFlags(cmd, &cfg)

	cmd.Flags().BoolVar(&cfg.OLAP, "olap", false, "Use OLAP to run the queries.")
	cmd.Flags().BoolVar(&cfg.XUnit, "xunit", false, "Get output in an xml file instead of errors directory")
	addInputTypeFlag(cmd, &inputType)
	addCSVConfigFlag(cmd, &csvConfig)

	return cmd
}

func tracerCmd() *cobra.Command {
	var cfg vttester.Config
	var inputType string
	csvConfig := data.NewEmptyCSVConfig(false, -1)

	cmd := &cobra.Command{
		Use:   "trace ",
		Short: "Runs the given workload and does a `vexplain trace` on all queries.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.TraceFile == "" {
				return errors.New("flag --trace-file is required when tracing")
			}
			cfg.Tests = args
			cfg.Compare = false
			loader, err := configureLoader(inputType, true, csvConfig)
			if err != nil {
				return err
			}
			cfg.Loader = loader

			return usageErr(cmd, vttester.Run(cfg))
		},
	}

	commonFlags(cmd, &cfg)
	addInputTypeFlag(cmd, &inputType)
	addCSVConfigFlag(cmd, &csvConfig)

	return cmd
}

func usageErr(cmd *cobra.Command, err error) error {
	if !errors.Is(err, vttester.WrongUsageError{}) {
		cmd.SilenceUsage = true
	}
	return err
}

func commonFlags(cmd *cobra.Command, cfg *vttester.Config) {
	cmd.Flags().StringVar(&cfg.LogLevel, "log-level", "error", "The log level of vt tester: info, warn, error, debug.")
	cmd.Flags().IntVar(&cfg.NumberOfShards, "number-of-shards", 0, "Number of shards to use for the sharded keyspace.")
	cmd.Flags().StringVar(&cfg.VschemaFile, "vschema", "", "Disable auto-vschema by providing your own vschema file. This cannot be used with either -vtexplain-vschema or -sharded.")
	cmd.Flags().StringVar(&cfg.VtExplainVschemaFile, "vtexplain-vschema", "", "Disable auto-vschema by providing your own vtexplain vschema file. This cannot be used with either -vschema or -sharded.")
	cmd.Flags().StringVar(&cfg.TraceFile, "trace-file", "", "Do a vexplain trace on all queries and store the output in the given file.")
	cmd.Flags().BoolVar(&cfg.Sharded, "sharded", false, "Run all tests on a sharded keyspace and using auto-vschema. This cannot be used with either -vschema or -vtexplain-vschema.")
	cmd.Flags().StringVar(&cfg.BackupDir, "backup-path", "", "Restore from backup before running the tester")
}
