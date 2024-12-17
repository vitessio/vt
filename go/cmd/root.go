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
	"os"

	"github.com/spf13/cobra"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := getRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func getRootCmd() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	root := &cobra.Command{
		Use:   "vt",
		Short: "Utils tools for testing, running and benchmarking Vitess.",
	}
	root.CompletionOptions.HiddenDefaultCmd = true

	root.AddCommand(summarizeCmd())
	root.AddCommand(testerCmd())
	root.AddCommand(tracerCmd())
	root.AddCommand(keysCmd())
	root.AddCommand(dbinfoCmd())
	root.AddCommand(transactionsCmd())
	root.AddCommand(planalyzeCmd())
	return root
}
