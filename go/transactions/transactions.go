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
	"slices"
	"sort"
	"strconv"
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

	TxSignature struct {
		Queries    []string
		Count      int
		Predicates []predicateInfo
	}

	predicateInfo struct {
		Table     string
		Col       string
		Op        sqlparser.ComparisonExprOperator
		Val       int
		Signature string
	}

	state struct {
		parser *sqlparser.Parser
		si     *keys.SchemaInfo
		mu     sync.Mutex
		txs    []TxSignature
	}
)

func (pi *predicateInfo) String() string {
	if pi.Signature != "" {
		return pi.Signature
	}
	pi.Signature = fmt.Sprintf("%s.%s %s %d", pi.Table, pi.Col, pi.Op.ToString(), pi.Val)
	return pi.Signature
}

func (tx *TxSignature) MarshalJSON() ([]byte, error) {
	// Transform Predicates to an array of strings
	predicateStrings := make([]string, len(tx.Predicates))
	for i, predicate := range tx.Predicates {
		predicateStrings[i] = predicate.String()
	}

	return json.Marshal(struct {
		Queries    []string
		Count      int
		Predicates []string
	}{
		Queries:    tx.Queries,
		Count:      tx.Count,
		Predicates: predicateStrings,
	})
}

func Run(cfg Config) {
	s := &state{
		parser: sqlparser.NewTestParser(),
		si:     getFakeSchema(),
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

	_ = data.ForeachSQLQuery(loader, func(query data.Query) error {
		stmt := s.parse(query.Query)
		if stmt == nil {
			return nil
		}
		switch stmt := stmt.(type) {
		case *sqlparser.Begin:
		case *sqlparser.Commit:
			// Commit seen, so we can yield the queries in the transaction
			connection := connections[query.ConnectionID]
			ch <- connection.Transaction
			connection.Transaction = nil
		case *sqlparser.Set:
			connection := connections[query.ConnectionID]
			connection.Autocommit = getAutocommitStatus(stmt, connection.Autocommit)
		default:
			if !sqlparser.IsDMLStatement(stmt) {
				return nil
			}
			connection, ok := connections[query.ConnectionID]
			if !ok {
				connection = &Connection{Autocommit: defaultAutocommit}
				connections[query.ConnectionID] = connection
			}
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
) predicateInfo {
	tableInfo, err := st.TableInfoForExpr(expr)
	if err != nil {
		panic(err)
	}
	table := tableInfo.GetVindexTable()
	if table == nil {
		panic("table not found")
	}
	return predicateInfo{
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

func getPredicates(e sqlparser.Expr, st *semantics.SemTable, n *normalizer) (predicates []predicateInfo) {
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
		tx := TxSignature{}
		for _, query := range queries {
			st, err := semantics.Analyze(query, "ks", s.si)
			if err != nil {
				panic(err)
			}

			switch query := query.(type) {
			case *sqlparser.Update:
				s.consumeUpdate(query, st, n, &tx)
			default:
				panic("not supported for now")
			}
		}
		s.addSignature(tx)
	}
}

func (tx *TxSignature) addPredicate(predicates []predicateInfo) {
loop:
	for i, predicate := range predicates {
		for _, txPred := range tx.Predicates {
			if txPred == predicate {
				continue loop
			}
		}

		tx.Predicates = append(tx.Predicates, predicates[i])
	}
}

func (s *state) consumeUpdate(query *sqlparser.Update, st *semantics.SemTable, n *normalizer, tx *TxSignature) {
	defer func() {
		tx.Queries = append(tx.Queries, sqlparser.String(query))
	}()

	// Normalize the AST our own way:
	// 	- Replace the value in SET by "v"
	// 	- Replace the literals found in where clause comparisons by the corresponding ID we got earlier
	for i, expr := range query.Exprs {
		query.Exprs[i] = &sqlparser.UpdateExpr{
			Name: expr.Name,
			Expr: sqlparser.NewArgument("v"),
		}
	}

	if query.Where == nil {
		return
	}

	// Find all predicates in the where clause that use a column and a literal
	// TODO: Implement support for join predicates
	tx.addPredicate(getPredicates(query.Where.Expr, st, n))

	var newWhere sqlparser.Where
	wheres := sqlparser.SplitAndExpression(nil, query.Where.Expr)
	for _, where := range wheres {
		switch cmp := where.(type) {
		case *sqlparser.ComparisonExpr:
			lhs, lhsOK := cmp.Left.(*sqlparser.Literal)
			rhs, rhsOK := cmp.Right.(*sqlparser.Literal)
			if !lhsOK && !rhsOK || lhsOK && rhsOK {
				newWhere.Expr = sqlparser.AndExpressions(newWhere.Expr)
				continue
			}

			var newCmp sqlparser.ComparisonExpr
			newCmp.Operator = cmp.Operator
			if lhsOK {
				id := n.normalize(lhs.Val)
				newCmp.Left = sqlparser.NewArgument(strconv.Itoa(id))
				newCmp.Right = cmp.Right
			} else {
				id := n.normalize(rhs.Val)
				newCmp.Right = sqlparser.NewArgument(strconv.Itoa(id))
				newCmp.Left = cmp.Left
			}
			newWhere.Expr = sqlparser.AndExpressions(newWhere.Expr, &newCmp)
		default:
			newWhere.Expr = sqlparser.AndExpressions(newWhere.Expr, where)
		}
	}
	query.Where = &newWhere
}

func (s *state) addSignature(tx TxSignature) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slices.Sort(tx.Queries)
	sort.Slice(tx.Predicates, func(i, j int) bool {
		return tx.Predicates[i]
	})

	s.txs = append(s.txs, tx)
}

func (s *state) run(out io.Writer, cfg Config) {
	defaultAutocommit := s.getAutocommitGuess(cfg)

	loader := cfg.Loader.Load(cfg.FileName)
	ch := make(chan []sqlparser.Statement, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	go s.consume(ch, &wg)
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

func getFakeSchema() *keys.SchemaInfo {
	// WIP: dummy data
	// TODO: Use real schema information data with the 'vt schema' JSON output
	si := &keys.SchemaInfo{
		Tables: map[string]keys.Columns{
			"tblA": {
				{Name: sqlparser.NewIdentifierCI("apa")},
				{Name: sqlparser.NewIdentifierCI("foo")},
				{Name: sqlparser.NewIdentifierCI("id")},
			},
			"tblB": {
				{Name: sqlparser.NewIdentifierCI("monkey")},
				{Name: sqlparser.NewIdentifierCI("bar")},
				{Name: sqlparser.NewIdentifierCI("id")},
			},
			"user": {
				{Name: sqlparser.NewIdentifierCI("id")},
				{Name: sqlparser.NewIdentifierCI("name")},
			},
			"user_extra": {
				{Name: sqlparser.NewIdentifierCI("user_id")},
				{Name: sqlparser.NewIdentifierCI("age")},
			},
		},
	}
	return si
}
