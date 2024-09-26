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

package vitess_tester

import (
	"fmt"

	"github.com/pingcap/errors"
	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/test/endtoend/utils"
	"vitess.io/vitess/go/vt/sqlparser"
)

type (
	// ComparingQueryRunner is a QueryRunner that compares the results of the queries between MySQL and Vitess
	ComparingQueryRunner struct {
		reporter          Reporter
		curr              utils.MySQLCompare
		handleCreateTable createTableHandler
		comparer          utils.MySQLCompare
	}
	createTableHandler func(create *sqlparser.CreateTable) func()
)

func newComparingQueryRunner(
	reporter Reporter,
	curr utils.MySQLCompare,
	handleCreateTable createTableHandler,
	comparer utils.MySQLCompare,
) *ComparingQueryRunner {
	return &ComparingQueryRunner{
		reporter:          reporter,
		curr:              curr,
		handleCreateTable: handleCreateTable,
		comparer:          comparer,
	}
}

func (nqr ComparingQueryRunner) runQuery(q query, expectedErrs bool) {
	// if t.vexplain != "" {
	// 	result, err := t.curr.VtConn.ExecuteFetch("vexplain "+t.vexplain+" "+query, -1, false)
	// 	t.vexplain = ""
	// 	if err != nil {
	// 		t.reporter.AddFailure(	// 		return
	// 	}
	//
	// 	t.reporter.AddInfo(fmt.Sprintf("VExplain Output:\n %s\n", result.Rows[0][0].ToString()))
	// }
	if err := nqr.execute(q, expectedErrs); err != nil {
		nqr.reporter.AddFailure(err)
	}
}

func (nqr *ComparingQueryRunner) execute(query query, expectedErrs bool) error {
	if len(query.Query) == 0 {
		return nil
	}

	parser := sqlparser.NewTestParser()
	ast, err := parser.Parse(query.Query)
	if err != nil {
		return err
	}

	// if sqlparser.IsDMLStatement(ast) && t.traceFile != nil && !t.expectedErrs {
	// 	// we don't want to run DMLs twice, so we just run them once while tracing
	// 	var errs []error
	// 	err := t.trace(query)
	// 	if err != nil {
	// 		errs = append(errs, err)
	// 	}
	//
	// 	// we need to run the DMLs on mysql as well
	// 	_, err = t.curr.MySQLConn.ExecuteFetch(query.Query, 10000, false)
	// 	if err != nil {
	// 		errs = append(errs, err)
	// 	}
	// 	return vterrors.Aggregate(errs)
	// }

	if err = nqr.executeStmt(query.Query, ast, expectedErrs); err != nil {
		return errors.Trace(errors.Errorf("run \"%v\" at line %d err %v", query.Query, query.Line, err))
	}
	// clear expected errors after we execute
	expectedErrs = false

	return nil

	// _, isDDL := ast.(sqlparser.DDLStatement)
	// if isDDL {
	// 	return nil
	// }

	// return t.trace(query)
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
		_, err := nqr.curr.ExecAllowAndCompareError(query, utils.CompareOptions{CompareColumnNames: true})
		if err == nil {
			// If we expected an error, but didn't get one, return an error
			return fmt.Errorf("expected error, but got none")
		}
	default:
		_ = nqr.curr.Exec(query)
	}
	return nil
}
