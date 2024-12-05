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

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/transactions"
)

func transactionsCmd() *cobra.Command {
	var inputType string

	cmd := &cobra.Command{
		Use:     "transactions ",
		Aliases: []string{"txs"},
		Short:   "Analyze transactions on a query log",
		Example: "vt transactions file.log",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg := transactions.Config{
				FileName: args[0],
			}

			loader, err := configureLoader(inputType, false, data.CSVConfig{})
			if err != nil {
				return err
			}
			cfg.Loader = loader

			transactions.Run(cfg)
			return nil
		},
	}

	addInputTypeFlag(cmd, &inputType)

	return cmd
}
