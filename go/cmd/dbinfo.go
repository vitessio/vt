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
	"vitess.io/vitess/go/mysql"

	"github.com/vitessio/vt/go/dbinfo"
)

func dbinfoCmd() *cobra.Command {
	var vtParams mysql.ConnParams

	cmd := &cobra.Command{
		Use:     "dbinfo ",
		Short:   "Loads info from the database including row counts",
		Example: "vt dbinfo",
		Args:    cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := dbinfo.Config{
				VTParams: vtParams,
			}

			return dbinfo.Run(cfg)
		},
	}

	cmd.Flags().StringVarP(&vtParams.Host, "host", "", "127.0.0.1", "Database host")
	cmd.Flags().IntVarP(&vtParams.Port, "port", "", 3306, "Database port")
	cmd.Flags().StringVarP(&vtParams.Uname, "user", "", "root", "Database user")
	cmd.Flags().StringVarP(&vtParams.Pass, "password", "", "", "Database password")
	cmd.Flags().StringVarP(&vtParams.DbName, "database", "", "sakila", "Database name")

	return cmd
}
