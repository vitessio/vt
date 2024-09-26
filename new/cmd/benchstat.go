package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vitessio/vitess-tester/src/cmd/vtbenchstat"
)

var benchstat = &cobra.Command{
	Use:     "benchstat old_file.json [new_file.json]",
	Short:   "Compares and analyses a trace output",
	Example: "vt benchstat old.json new.json",
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		vtbenchstat.Run(args)
	},
}
