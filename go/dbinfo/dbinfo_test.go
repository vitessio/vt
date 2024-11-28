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

package dbinfo

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/test/endtoend/utils"
)

func TestDBInfoLoad(t *testing.T) {
	si, err := Load("../testdata/sakila-dbinfo.json")
	require.NoError(t, err)
	require.NotNil(t, si)

	t.Run("validateTableInfo", func(t *testing.T) {
		require.NotEmpty(t, si.Tables)
		require.Len(t, si.Tables, 23)
		var tables []string
		for _, table := range si.Tables {
			tables = append(tables, table.Name)
		}
		require.Contains(t, tables, "actor")
		require.NotContains(t, tables, "foo")
		for _, table := range si.Tables {
			require.NotEmpty(t, table.Name)
			switch table.Name {
			case "language":
				require.Equal(t, 6, table.Rows)
			case "film":
				require.Equal(t, 1000, table.Rows)
			}
		}
	})

	t.Run("validateGlobalVariables", func(t *testing.T) {
		require.NotEmpty(t, si.GlobalVariables)
		require.Len(t, *si.GlobalVariables, 3)
		expected := map[string]string{
			"binlog_format":    "ROW",
			"binlog_row_image": "FULL",
			"log_bin":          "ON",
		}
		require.EqualValues(t, expected, *si.GlobalVariables)
	})
}

func TestDBInfoGet(t *testing.T) {
	clusterInstance := cluster.NewCluster("zone1", "127.0.0.1")
	clusterInstance.Teardown()

	err := clusterInstance.StartTopo()
	require.NoError(t, err)

	schemaFile := "../testdata/sakila-schema-ddls.sql"
	schemaBytes, err := os.ReadFile(schemaFile)
	require.NoError(t, err)

	dataFile := "../testdata/sakila-data.sql"
	dataBytes, err := os.ReadFile(dataFile)
	require.NoError(t, err)

	cp, cancel, err := utils.NewMySQL(clusterInstance, "sakila")
	require.NoError(t, err)
	defer cancel()

	t.Run("createSchema", func(t *testing.T) {
		conn, err := mysql.Connect(context.Background(), &cp)
		require.NoError(t, err)
		qr, err := conn.ExecuteFetch("show databases", 1000, false)
		require.NoError(t, err)
		for _, row := range qr.Rows {
			if row[0].ToString() == "sakila" {
				err := conn.ExecuteFetchMultiDrain(string(schemaBytes))
				require.NoError(t, err)
				err = conn.ExecuteFetchMultiDrain(string(dataBytes))
				require.NoError(t, err)
			}
		}
	})

	dbh := NewDBHelper(&cp)

	t.Run("getTableSizes", func(t *testing.T) {
		ts, err := dbh.getTableSizes()
		require.NoError(t, err)
		require.NotNil(t, ts)
		require.Len(t, ts, 16)
		require.Equal(t, 6, ts["language"])
		require.Equal(t, 1000, ts["film"])
	})

	t.Run("getColumnInfo", func(t *testing.T) {
		tc, err := dbh.getColumnInfo()
		require.NoError(t, err)
		require.NotNil(t, tc)
		require.Len(t, tc, 16)

		require.Len(t, tc["language"], 3)
		cols := make(map[string]*TableColumn)
		for _, column := range tc["language"] {
			cols[column.Name] = column
		}
		for _, column := range []string{"language_id", "name", "last_update"} {
			require.Contains(t, cols, column)
		}
		colLanguageID := cols["language_id"]
		require.Equal(t, "tinyint", colLanguageID.Type)
		require.Equal(t, "pri", colLanguageID.KeyType)
		require.False(t, colLanguageID.IsNullable)
		require.Equal(t, "auto_increment", colLanguageID.Extra)

		colName := cols["name"]
		require.Equal(t, "char", colName.Type)
		require.Empty(t, colName.KeyType)
		require.False(t, colName.IsNullable)

		colLastUpdate := cols["last_update"]
		require.Equal(t, "timestamp", colLastUpdate.Type)
		require.Empty(t, colLastUpdate.KeyType)
		require.False(t, colLastUpdate.IsNullable)
		require.Equal(t, "default_generated on update current_timestamp", colLastUpdate.Extra)
	})

	t.Run("getGlobalVariables", func(t *testing.T) {
		gv, err := dbh.getGlobalVariables()
		require.NoError(t, err)
		require.NotNil(t, gv)
		require.NotEmpty(t, (gv))
	})
}
