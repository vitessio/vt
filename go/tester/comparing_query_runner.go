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
	"fmt"
	"github.com/vitessio/vitess-tester/go/tools"

	"github.com/pingcap/errors"
	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/test/endtoend/utils"
	"vitess.io/vitess/go/vt/sqlparser"
)

type (
	// ComparingQueryRunner is a QueryRunner that compares the results of the queries between MySQL and Vitess
	ComparingQueryRunner struct {
		reporter          Reporter
		handleCreateTable CreateTableHandler
		comparer          utils.MySQLCompare
	}
	CreateTableHandler          func(create *sqlparser.CreateTable) func()
	ComparingQueryRunnerFactory struct{}
)

func (f ComparingQueryRunnerFactory) Close() {}

func (f ComparingQueryRunnerFactory) NewQueryRunner(reporter Reporter, handleCreateTable CreateTableHandler, comparer utils.MySQLCompare) QueryRunner {
	return newComparingQueryRunner(reporter, handleCreateTable, comparer)
}

func newComparingQueryRunner(
	reporter Reporter,
	handleCreateTable CreateTableHandler,
	comparer utils.MySQLCompare,
) *ComparingQueryRunner {
	return &ComparingQueryRunner{
		reporter:          reporter,
		handleCreateTable: handleCreateTable,
		comparer:          comparer,
	}
}

func (nqr ComparingQueryRunner) runQuery(q tools.Query, expectedErrs bool, ast sqlparser.Statement) error {
	return nqr.execute(q, expectedErrs, ast)
}

func (nqr *ComparingQueryRunner) execute(query tools.Query, expectedErrs bool, ast sqlparser.Statement) error {
	if len(query.Query) == 0 {
		return nil
	}

	if err := nqr.executeStmt(query.Query, ast, expectedErrs); err != nil {
		return errors.Trace(errors.Errorf("run \"%v\" at line %d err %v", query.Query, query.Line, err))
	}
	// clear expected errors after we execute
	expectedErrs = false

	return nil
}

func (nqr *ComparingQueryRunner) executeStmt(query string, ast sqlparser.Statement, expectedErrs bool) (err error) {
	_, commentOnly := ast.(*sqlparser.CommentOnly)
	if commentOnly {
		return nil
	}

	log.Debugf("executeStmt: %s", query)
	create, isCreateStatement := ast.(*sqlparser.CreateTable)
	if isCreateStatement && !expectedErrs {
		closer := nqr.handleCreateTable(create)
		defer func() {
			if err == nil {
				closer()
			}
		}()
	}

	switch {
	case expectedErrs:
		_, err := nqr.comparer.ExecAllowAndCompareError(query, utils.CompareOptions{CompareColumnNames: true})
		if err == nil {
			// If we expected an error, but didn't get one, return an error
			return fmt.Errorf("expected error, but got none")
		}
	default:
		_ = nqr.comparer.Exec(query)
	}
	return nil
}
