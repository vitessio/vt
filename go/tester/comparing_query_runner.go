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

	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/test/endtoend/utils"
	"vitess.io/vitess/go/vt/sqlparser"

	"github.com/vitessio/vt/go/data"
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

func (nqr ComparingQueryRunner) runQuery(q data.Query, expectedErrs bool, cfg QueryRunConfig) error {
	if !cfg.vitess && !cfg.mysql {
		return fmt.Errorf("both vitess and mysql are false")
	}
	return nqr.execute(q, expectedErrs, cfg.ast, cfg.vitess, cfg.mysql)
}

func (nqr *ComparingQueryRunner) execute(query data.Query, expectedErrs bool, ast sqlparser.Statement, vitess bool, mysql bool) error {
	if len(query.Query) == 0 {
		return nil
	}

	defer func() {
		// clear expected errors after we execute
		expectedErrs = false
	}()

	if err := nqr.executeStmt(query.Query, ast, expectedErrs, vitess, mysql); err != nil {
		return fmt.Errorf("run \"%v\" at line %d err %v", query.Query, query.Line, err)
	}

	return nil
}

func (nqr *ComparingQueryRunner) executeStmt(query string, ast sqlparser.Statement, expectedErrs bool, vitess bool, mysql bool) (err error) {
	_, commentOnly := ast.(*sqlparser.CommentOnly)
	if commentOnly {
		return nil
	}

	log.Debugf("executeStmt: %s", query)
	create, isCreateStatement := ast.(*sqlparser.CreateTable)
	if isCreateStatement && !expectedErrs && vitess {
		closer := nqr.handleCreateTable(create)
		defer func() {
			if err == nil {
				closer()
			}
		}()
	}

	switch {
	case expectedErrs:
		err := nqr.execAndExpectErr(query, vitess, mysql)
		if err != nil {
			nqr.reporter.AddFailure(err)
		}
	default:
		var err error
		switch {
		case vitess && !mysql:
			_, err = nqr.comparer.VtConn.ExecuteFetch(query, 1000, true)
		case mysql && !vitess:
			_, err = nqr.comparer.MySQLConn.ExecuteFetch(query, 1000, true)
		case mysql:
			nqr.comparer.Exec(query)
		}
		if err != nil {
			nqr.reporter.AddFailure(err)
		}
	}
	return nil
}

func (nqr *ComparingQueryRunner) execAndExpectErr(query string, vitess bool, mysql bool) error {
	var err error
	switch {
	case vitess && !mysql:
		_, err = nqr.comparer.VtConn.ExecuteFetch(query, 1000, true)
	case mysql && !vitess:
		_, err = nqr.comparer.MySQLConn.ExecuteFetch(query, 1000, true)
	case mysql:
		_, err = nqr.comparer.ExecAllowAndCompareError(query, utils.CompareOptions{CompareColumnNames: true})
		return err
	}

	if err == nil {
		// If we expected an error, but didn't get one, return an error
		return fmt.Errorf("expected error, but got none")
	}
	return nil
}
