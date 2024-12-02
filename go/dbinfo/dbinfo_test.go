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
		require.Len(t, si.GlobalVariables, 4)
		expected := map[string]string{
			"binlog_format":    "ROW",
			"binlog_row_image": "FULL",
			"gtid_mode":        "OFF",
			"log_bin":          "ON",
		}
		require.EqualValues(t, expected, si.GlobalVariables)
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

	t.Run("create schema", func(t *testing.T) {
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

	t.Run("table sizes", func(t *testing.T) {
		ts, err := dbh.getTableSizes()
		require.NoError(t, err)
		require.Len(t, ts, 16)
		require.Equal(t, 6, ts["language"])
		require.Equal(t, 1000, ts["film"])
	})

	t.Run("column info", func(t *testing.T) {
		tc, err := dbh.getColumnInfo()
		require.NoError(t, err)
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

	t.Run("global variables", func(t *testing.T) {
		gv, err := dbh.getGlobalVariables()
		require.NoError(t, err)
		require.NotEmpty(t, gv)
	})

	t.Run("primary keys", func(t *testing.T) {
		pks, err := dbh.getPrimaryKeys()
		require.NoError(t, err)
		require.Len(t, pks, 16)
		want := map[string][]string{
			"actor":         {"actor_id"},
			"film":          {"film_id"},
			"language":      {"language_id"},
			"film_category": {"film_id", "category_id"},
			"film_actor":    {"actor_id", "film_id"},
		}
		for tableName, Columns := range want {
			pk, ok := pks[tableName]
			require.True(t, ok)
			require.Equal(t, Columns, pk.columns)
		}
	})

	t.Run("indexes", func(t *testing.T) {
		idxs, err := dbh.getIndexes()
		require.NoError(t, err)
		require.Len(t, idxs, 16)
		idx, ok := idxs["film_actor"]
		require.True(t, ok)
		require.Len(t, idx.indexes, 2)
		require.Equal(t, "idx_fk_film_id", idx.indexes["idx_fk_film_id"].Name)
		require.Equal(t, []string{"film_id"}, idx.indexes["idx_fk_film_id"].Columns)
		idx, ok = idxs["rental"]
		require.True(t, ok)
		require.Len(t, idx.indexes, 5)
		require.Equal(t, "rental_date", idx.indexes["rental_date"].Name)
		require.Equal(t, []string{"rental_date", "inventory_id", "customer_id"}, idx.indexes["rental_date"].Columns)
		require.Equal(t, "PRIMARY", idx.indexes["PRIMARY_KEY"].Name)
		require.Equal(t, []string{"rental_id"}, idx.indexes["PRIMARY_KEY"].Columns)
	})

	t.Run("foreign keys", func(t *testing.T) {
		fks, err := dbh.getForeignKeys()
		require.NoError(t, err)
		require.Len(t, fks, 11)
		fk, ok := fks["city"]
		require.True(t, ok)
		require.Len(t, fk, 1)
		require.Equal(t, "country_id", fk[0].ColumnName)
		require.Equal(t, "fk_city_country", fk[0].ConstraintName)
		require.Equal(t, "country", fk[0].ReferencedTableName)
		require.Equal(t, "country_id", fk[0].ReferencedColumnName)

		fk, ok = fks["store"]
		require.True(t, ok)
		require.Len(t, fk, 2)
		require.Equal(t, "address_id", fk[0].ColumnName)
		require.Equal(t, "fk_store_address", fk[0].ConstraintName)
		require.Equal(t, "address", fk[0].ReferencedTableName)
		require.Equal(t, "address_id", fk[0].ReferencedColumnName)
		require.Equal(t, "manager_staff_id", fk[1].ColumnName)
		require.Equal(t, "fk_store_staff", fk[1].ConstraintName)
		require.Equal(t, "staff", fk[1].ReferencedTableName)
		require.Equal(t, "staff_id", fk[1].ReferencedColumnName)
	})
}
