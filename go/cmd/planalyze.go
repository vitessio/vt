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
	"github.com/spf13/cobra"

	"github.com/vitessio/vt/go/planalyze"
)

func planalyzeCmd() *cobra.Command {
	var cfg planalyze.Config

	cmd := &cobra.Command{
		Use:     "planalyze",
		Short:   "Analyze the query plans using the keys output",
		Long:    "Analyze the query plans. The report will report how many queries fall into one of the four categories: `passthrough`, `simple-routed`, `complex`, `unplannable`.",
		Example: "vt planalyze --vcshema file.vschema keys-log.json",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return planalyze.Run(cfg, args[0])
		},
	}

	cmd.Flags().StringVar(&cfg.VSchemaFile, "vschema", "", "Supply the vschema in a format that can contain multiple keyspaces. This cannot be used with -vtexplain-vschema.")
	cmd.Flags().StringVar(&cfg.VtExplainVschemaFile, "vtexplain-vschema", "", "Supply the vschema in a format that contains a single keyspace")

	return cmd
}
