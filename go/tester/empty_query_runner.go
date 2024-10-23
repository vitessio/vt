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
	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/tester/state"
	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/test/endtoend/utils"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"
)

type Empty struct{}

func (e *Empty) NewQueryRunner(Reporter, CreateTableHandler, utils.MySQLCompare, *cluster.LocalProcessCluster, *vindexes.VSchema) QueryRunner {
	return e
}

func (e *Empty) Close() {}

func (e *Empty) runQuery(data.Query, sqlparser.Statement, *state.State) error { return nil }
