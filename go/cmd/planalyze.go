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
		Short:   "Analyze the query plan",
		Example: "vt planalyze --vcshema-file file.vschema keys-log.json",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return planalyze.Run(cfg, args[0])
		},
	}

	cmd.Flags().StringVar(&cfg.VSchemaFile, "vschema", "", "Disable auto-vschema by providing your own vschema file. This cannot be used with either -vtexplain-vschema or -sharded.")
	cmd.Flags().StringVar(&cfg.VtExplainVschemaFile, "vtexplain-vschema", "", "Disable auto-vschema by providing your own vtexplain vschema file. This cannot be used with either -vschema or -sharded.")

	return cmd
}
