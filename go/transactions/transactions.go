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

package transactions

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/semantics"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
)

type (
	Config struct {
		FileName string
		Loader   data.Loader
	}

	Connection struct {
		Transaction []sqlparser.Statement

		Autocommit bool
	}

	state struct {
		parser *sqlparser.Parser
		si     *keys.SchemaInfo
		mu     sync.Mutex
		txs    *txSignatureMap
	}
)

func Run(cfg Config) {
	s := &state{
		parser: sqlparser.NewTestParser(),
		si:     &keys.SchemaInfo{},
		txs:    newTxSignatureMap(),
	}
	s.run(os.Stdout, cfg)
}

func getAutocommitStatus(set *sqlparser.Set, oldState bool) bool {
	for _, expr := range set.Exprs {
		if expr.Var.Name.Lowered() == "autocommit" {
			val, ok := expr.Expr.(*sqlparser.Literal)
			if !ok {
				continue
			}
			str := strings.ToLower(val.Val)
			if str == "1" || str == "on" || str == "true" {
				return true
			}
			return false
		}
	}
	return oldState
}

func (s *state) parse(q string) sqlparser.Statement {
	stmt, err := s.parser.Parse(q)
	if err != nil {
		return nil
	}
	return stmt
}

func (s *state) startProducing(loader data.IteratorLoader, defaultAutocommit bool, ch chan<- []sqlparser.Statement) {
	connections := map[int]*Connection{}
	getConn := func(id int) *Connection {
		connection, ok := connections[id]
		if !ok {
			connection = &Connection{Autocommit: defaultAutocommit}
			connections[id] = connection
		}
		return connection
	}
	_ = data.ForeachSQLQuery(loader, func(query data.Query) error {
		stmt := s.parse(query.Query)
		if stmt == nil {
			return nil
		}
		switch stmt := stmt.(type) {
		case *sqlparser.Begin:
		case *sqlparser.Commit:
			// Commit seen, so we can yield the queries in the transaction
			connection := getConn(query.ConnectionID)
			if connection.Transaction == nil {
				return nil
			}
			ch <- connection.Transaction
			connection.Transaction = nil
		case *sqlparser.Set:
			conn := getConn(query.ConnectionID)
			conn.Autocommit = getAutocommitStatus(stmt, defaultAutocommit)
		default:
			if !sqlparser.IsDMLStatement(stmt) {
				return nil
			}
			connection := getConn(query.ConnectionID)
			if connection.Autocommit {
				ch <- []sqlparser.Statement{stmt}
			} else {
				connection.Transaction = append(connection.Transaction, stmt)
			}
		}
		return nil
	})
}

func exprToString(expr sqlparser.Expr) string {
	if v, ok := expr.(*sqlparser.Literal); ok {
		return v.Val
	}
	return ""
}

func createPredicateInfo(
	st *semantics.SemTable,
	expr *sqlparser.ColName,
	op sqlparser.ComparisonExprOperator,
	value string,
	n *normalizer,
) PredicateInfo {
	tableInfo, err := st.TableInfoForExpr(expr)
	if err != nil {
		panic(err)
	}
	table := tableInfo.GetVindexTable()
	if table == nil {
		panic("table not found")
	}
	return PredicateInfo{
		Table: table.Name.String(),
		Col:   expr.Name.String(),
		Op:    op,
		Val:   n.normalize(value),
	}
}

type normalizer struct {
	m    map[string]int
	next int
}

func (n *normalizer) normalize(s string) int {
	v, ok := n.m[s]
	if ok {
		return v
	}
	id := n.next
	n.m[s] = id
	n.next++
	return id
}

func getPredicates(e sqlparser.Expr, st *semantics.SemTable, n *normalizer) (predicates []PredicateInfo) {
	// TODO: Implement support for join predicates
	for _, predicate := range sqlparser.SplitAndExpression(nil, e) {
		cmp, ok := predicate.(*sqlparser.ComparisonExpr)
		if !ok {
			continue
		}

		lhs, lhsOK := cmp.Left.(*sqlparser.ColName)
		rhs, rhsOK := cmp.Right.(*sqlparser.ColName)

		if rhsStr := exprToString(cmp.Right); lhsOK && rhsStr != "" {
			predicates = append(predicates, createPredicateInfo(st, lhs, cmp.Operator, rhsStr, n))
		}

		if lhsStr := exprToString(cmp.Left); rhsOK && lhsStr != "" {
			switchedOp, ok := cmp.Operator.SwitchSides()
			if ok {
				predicates = append(predicates, createPredicateInfo(st, rhs, switchedOp, lhsStr, n))
			}
		}
	}

	return
}

func (s *state) consume(ch <-chan []sqlparser.Statement, wg *sync.WaitGroup) {
	defer wg.Done()
	for queries := range ch {
		n := &normalizer{m: make(map[string]int)}
		tx := &Signature{}
		for _, query := range queries {
			st, err := semantics.Analyze(query, "ks", s.si)
			if err != nil {
				panic(err)
			}

			switch query := query.(type) {
			case *sqlparser.Update:
				s.consumeUpdate(query, st, n, tx)
			case *sqlparser.Delete:
				s.consumeDelete(query, st, n, tx)
			}
		}
		s.addSignature(tx)
	}
}

func (s *state) consumeUpdate(query *sqlparser.Update, st *semantics.SemTable, n *normalizer, tx *Signature) {
	// Find all predicates in the where clause that use a column and a literal
	var predicates []PredicateInfo
	if query.Where != nil {
		predicates = getPredicates(query.Where.Expr, st, n)
	}

	updatedColumns := make([]string, 0, len(query.Exprs))
	for _, expr := range query.Exprs {
		updatedColumns = append(updatedColumns, sqlparser.String(expr.Name.Name))
	}

	if len(query.TableExprs) != 1 {
		// TODO: Implement support for multi-table updates
		panic("multi-table updates not supported")
	}

	tx.Queries = append(tx.Queries, Query{
		Op:             "update",
		AffectedTable:  sqlparser.String(query.TableExprs[0]),
		UpdatedColumns: updatedColumns,
		Predicates:     predicates,
	})
}

func (s *state) consumeDelete(del *sqlparser.Delete, st *semantics.SemTable, n *normalizer, tx *Signature) {
	var predicates []PredicateInfo
	if del.Where != nil {
		predicates = getPredicates(del.Where.Expr, st, n)
	}
	if len(del.TableExprs) != 1 {
		// TODO: Implement support for multi-table deletes
		panic("multi-table updates not supported")
	}

	tx.Queries = append(tx.Queries, Query{
		Op:            "delete",
		AffectedTable: sqlparser.String(del.TableExprs[0]),
		Predicates:    predicates,
	})
}

func (s *state) addSignature(tx *Signature) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.txs.Add(tx)
}

func (s *state) run(out io.Writer, cfg Config) {
	defaultAutocommit := s.getAutocommitGuess(cfg)

	loader := cfg.Loader.Load(cfg.FileName)
	ch := make(chan []sqlparser.Statement, 1000)

	noOfConsumers := 1
	var wg sync.WaitGroup
	for range noOfConsumers {
		wg.Add(1)
		go s.consume(ch, &wg)
	}

	go func() {
		s.startProducing(loader, defaultAutocommit, ch)
		close(ch)
	}()

	wg.Wait()

	txsJSON, err := json.MarshalIndent(s.txs, "", "  ")
	if err != nil {
		panic(err)
	}
	_, _ = fmt.Fprintf(out, "%s\n", string(txsJSON))
}

func (s *state) getAutocommitGuess(cfg Config) bool {
	// Figure out if autocommit is enabled
	// If we see:
	// 1. BEGIN we can assume autocommit is disabled
	// 2. COMMIT and no BEGIN we can assume autocommit is enabled
	// 3. ROLLBACK and no BEGIN we can assume autocommit is enabled
	// 4. SET autocommit = 1/0
	count := 1000
	defaultAutocommit := true
	loader := cfg.Loader.Load(cfg.FileName)
	defer func() {
		err := loader.Close()
		if err != nil {
			panic(err.Error())
		}
	}()
	_ = data.ForeachSQLQuery(loader, func(query data.Query) error {
		count--
		if count == 0 {
			// enough already. we'll assume autocommit is enabled because that is the default
			return io.EOF
		}

		stmt := s.parse(query.Query)
		if stmt == nil {
			return nil
		}

		switch stmt.(type) {
		case *sqlparser.Begin:
			// BEGIN seen, so autocommit is disabled
			return io.EOF
		case *sqlparser.Commit:
			defaultAutocommit = false
			// no BEGIN seen, so autocommit is disabled
			return io.EOF
		}

		return nil
	})
	return defaultAutocommit
}
