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
	"vitess.io/vitess/go/test/endtoend/cluster"
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
		cluster           *cluster.LocalProcessCluster
	}
	CreateTableHandler          func(create *sqlparser.CreateTable) func()
	ComparingQueryRunnerFactory struct{}
)

func (f ComparingQueryRunnerFactory) Close() {}

func (f ComparingQueryRunnerFactory) NewQueryRunner(reporter Reporter, handleCreateTable CreateTableHandler, comparer utils.MySQLCompare, cluster *cluster.LocalProcessCluster, table func(name string) (ks string, err error)) QueryRunner {
	return newComparingQueryRunner(reporter, handleCreateTable, comparer, cluster)
}

func newComparingQueryRunner(
	reporter Reporter,
	handleCreateTable CreateTableHandler,
	comparer utils.MySQLCompare,
	cluster *cluster.LocalProcessCluster,
) *ComparingQueryRunner {
	return &ComparingQueryRunner{
		reporter:          reporter,
		handleCreateTable: handleCreateTable,
		comparer:          comparer,
		cluster:           cluster,
	}
}

func (nqr ComparingQueryRunner) runQuery(q data.Query, expectedErrs bool, cfg QueryRunConfig) error {
	return nqr.execute(q, expectedErrs, cfg)
}

func (nqr *ComparingQueryRunner) execute(query data.Query, expectedErrs bool, cfg QueryRunConfig) error {
	if len(query.Query) == 0 {
		return nil
	}

	defer func() {
		// clear expected errors after we execute
		expectedErrs = false
	}()

	if err := nqr.executeStmt(query.Query, cfg, expectedErrs); err != nil {
		return fmt.Errorf("run \"%v\" at line %d err %v", query.Query, query.Line, err)
	}

	return nil
}

func (nqr *ComparingQueryRunner) executeStmt(query string, cfg QueryRunConfig, expectedErrs bool) (err error) {
	_, commentOnly := cfg.ast.(*sqlparser.CommentOnly)
	if commentOnly {
		return nil
	}

	log.Debugf("executeStmt: %s", query)
	create, isCreateStatement := cfg.ast.(*sqlparser.CreateTable)
	if isCreateStatement && !expectedErrs && cfg.vitess {
		closer := nqr.handleCreateTable(create)
		defer func() {
			if err == nil {
				closer()
			}
		}()
	}

	switch {
	case expectedErrs:
		err := nqr.execAndExpectErr(query)
		if err != nil {
			nqr.reporter.AddFailure(err)
		}
	default:
		var err error
		switch {
		case cfg.reference:
			return nqr.executeReference(query, cfg.ast)
		case cfg.mysql && cfg.vitess:
			nqr.comparer.Exec(query)
		case cfg.vitess:
			_, err = nqr.comparer.VtConn.ExecuteFetch(query, 1000, true)
		case cfg.mysql:
			_, err = nqr.comparer.MySQLConn.ExecuteFetch(query, 1000, true)
		}
		if err != nil {
			nqr.reporter.AddFailure(err)
		}
	}
	return nil
}

func (nqr *ComparingQueryRunner) execAndExpectErr(query string) error {
	_, err := nqr.comparer.ExecAllowAndCompareError(query, utils.CompareOptions{CompareColumnNames: true})
	if err == nil {
		// If we expected an error, but didn't get one, return an error
		return fmt.Errorf("expected error, but got none")
	}
	return nil
}

func (nqr *ComparingQueryRunner) executeReference(query string, ast sqlparser.Statement) error {
	_, err := nqr.comparer.MySQLConn.ExecuteFetch(query, 1000, true)
	if err != nil {
		return err
	}

	tables := sqlparser.ExtractAllTables(ast)
	if len(tables) != 1 {
		return fmt.Errorf("expected exactly one table in the query, got %d", len(tables))
	}

	tableName := tables[0]

	tbl, err := vschema.FindTable("" /*empty means global search*/, tableName)
	if err != nil {
		return err
	}

	for _, ks := range nqr.cluster.Keyspaces {
		if ks.Name == tbl.Keyspace.Name {
			for _, shard := range ks.Shards {
				_, err := nqr.comparer.VtConn.ExecuteFetch(fmt.Sprintf("use `%s/%s`", ks.Name, shard.Name), 1000, true)
				if err != nil {
					return fmt.Errorf("error setting keyspace/shard: %w", err)
				}
				_, err = nqr.comparer.VtConn.ExecuteFetch(query, 1000, true)
				if err != nil {
					return fmt.Errorf("error executing query on vtgate: %w", err)
				}
			}
			q := fmt.Sprintf("use %s", ks.Name)
			_, err = nqr.comparer.VtConn.ExecuteFetch(q, 1000, true)
			if err != nil {
				return fmt.Errorf("error setting keyspace: %s %w", q, err)
			}
		}
	}

	return nil
}
