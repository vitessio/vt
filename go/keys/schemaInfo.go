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
	"vitess.io/vitess/go/mysql/collations"
	"vitess.io/vitess/go/vt/key"
	"vitess.io/vitess/go/vt/proto/topodata"
	vschemapb "vitess.io/vitess/go/vt/proto/vschema"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtenv"
	"vitess.io/vitess/go/vt/vtgate/semantics"
	"vitess.io/vitess/go/vt/vtgate/vindexes"
)

var _ semantics.SchemaInformation = (*SchemaInfo)(nil)

type (
	// SchemaInfo is a simple implementation of semantics.SchemaInformation
	// It will claim that any table that is asked for is present in the schema, with no columns specified and authoratative columns set to false
	// it has a createTableHandler that can be used to populate the schema if the
	// query log contains the CREATE TABLE statements
	SchemaInfo struct {
		KsName string
		Tables map[string]Columns
	}

	Columns []vindexes.Column
)

func (s *SchemaInfo) handleCreateTable(create *sqlparser.CreateTable) {
	columns := make(Columns, 0, len(create.TableSpec.Columns))
	for _, col := range create.TableSpec.Columns {
		columns = append(columns, vindexes.Column{
			Name: col.Name,
			Type: col.Type.SQLType(),
		})
	}
	s.Tables[create.Table.Name.String()] = columns
}

func (s *SchemaInfo) FindTableOrVindex(tablename sqlparser.TableName) (*vindexes.Table, vindexes.Vindex, string, topodata.TabletType, key.Destination, error) {
	var tbl *vindexes.Table
	ks := tablename.Qualifier.String()
	if ks == "" {
		ks = s.KsName
	}

	if !tablename.Qualifier.NotEmpty() || tablename.Qualifier.String() == s.KsName {
		// This is a table from our keyspace. We should be able to find it
		columns, found := s.Tables[tablename.Name.String()]
		if found {
			tbl = &vindexes.Table{
				Name:                    tablename.Name,
				Keyspace:                &vindexes.Keyspace{Name: s.KsName, Sharded: true},
				Columns:                 columns,
				ColumnListAuthoritative: true,
			}
		}
	}

	if tbl == nil {
		// This is a table from another keyspace, or we couldn't find it in our keyspace
		tbl = &vindexes.Table{
			Name:                    tablename.Name,
			Keyspace:                &vindexes.Keyspace{Name: ks, Sharded: true},
			ColumnListAuthoritative: false,
		}
	}

	return tbl, nil, ks, topodata.TabletType_REPLICA, nil, nil
}

func (s *SchemaInfo) ConnCollation() collations.ID {
	return collations.CollationBinaryID
}

func (s *SchemaInfo) Environment() *vtenv.Environment {
	return vtenv.NewTestEnv()
}

func (s *SchemaInfo) ForeignKeyMode(string) (vschemapb.Keyspace_ForeignKeyMode, error) {
	return vschemapb.Keyspace_unmanaged, nil
}

func (s *SchemaInfo) GetForeignKeyChecksState() *bool {
	return nil
}

func (s *SchemaInfo) KeyspaceError(string) error {
	return nil
}

func (s *SchemaInfo) GetAggregateUDFs() []string {
	return nil // TODO: maybe this should be a flag?
}

func (s *SchemaInfo) FindMirrorRule(sqlparser.TableName) (*vindexes.MirrorRule, error) {
	return nil, nil
}
