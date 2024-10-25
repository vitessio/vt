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

package tester

import (
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/test/endtoend/utils"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/tester/state"
)

type NullQueryRunner struct {
	VtConn            *mysql.Conn
	handleCreateTable CreateTableHandler
}

type NullQueryRunnerFactory struct{}

func (NullQueryRunnerFactory) Close() {}

func (NullQueryRunnerFactory) NewQueryRunner(_ Reporter, handleCreateTable CreateTableHandler, mcmp utils.MySQLCompare, _ *cluster.LocalProcessCluster, _ *vindexes.VSchema) QueryRunner {
	return &NullQueryRunner{
		handleCreateTable: handleCreateTable,
		VtConn:            mcmp.VtConn,
	}
}

func (nqr *NullQueryRunner) runQuery(q data.Query, ast sqlparser.Statement, state *state.State) error {
	create, isCreateStatement := ast.(*sqlparser.CreateTable)
	if isCreateStatement && !state.IsErrorExpectedSet() && state.RunOnVitess() {
		closer := nqr.handleCreateTable(create)
		closer()
	}

	_, err := nqr.VtConn.ExecuteFetch(q.Query, 1000, false)
	return err
}
