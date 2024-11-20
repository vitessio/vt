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

package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"vitess.io/vitess/go/slice"
	"vitess.io/vitess/go/test/utils"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"
)

func TestSchemaInfo(t *testing.T) {
	parser := sqlparser.NewTestParser()

	si := &SchemaInfo{Tables: make(map[string]Columns)}

	ast, err := parser.Parse(`CREATE TABLE IF NOT EXISTS warehouse (
	w_id INT NOT NULL,
	w_name VARCHAR(10),
	w_street_1 VARCHAR(20),
	w_state CHAR(2),
	w_zip CHAR(9),
	w_tax DECIMAL(4, 4),
	w_ytd DECIMAL(12, 2),
	PRIMARY KEY (w_id)
)`)
	require.NoError(t, err)
	create, ok := ast.(*sqlparser.CreateTable)
	require.True(t, ok, "not a create table statement")
	si.handleCreateTable(create)

	tableName := sqlparser.TableName{Name: sqlparser.NewIdentifierCS("warehouse")}
	table, _, _, _, _, err := si.FindTableOrVindex(tableName)
	require.NoError(t, err)

	colNames := slice.Map(table.Columns, func(c vindexes.Column) string {
		return c.Name.String()
	})
	utils.MustMatch(t, []string{"w_id", "w_name", "w_street_1", "w_state", "w_zip", "w_tax", "w_ytd"}, colNames)

	colTypes := slice.Map(table.Columns, func(c vindexes.Column) string {
		return c.Type.String()
	})
	utils.MustMatch(t, []string{"INT32", "VARCHAR", "VARCHAR", "CHAR", "CHAR", "DECIMAL", "DECIMAL"}, colTypes)
}
