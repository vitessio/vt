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
	"strconv"
	"strings"

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
		Table string
		Col   string
		Op    sqlparser.ComparisonExprOperator
		Val   string
	}
)

func (pi predicateInfo) String() string {
	return fmt.Sprintf("%s.%s %s %s", pi.Table, pi.Col, pi.Op.ToString(), pi.Val)
}

func (tx TxSignature) MarshalJSON() ([]byte, error) {
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
	run(os.Stdout, cfg)
}

//nolint:funlen,gocognit,gocyclo,cyclop // this is dirty WIP
func run(out io.Writer, cfg Config) {
	defaultAutocommit := GetAutocommitGuess(cfg)
	transactions := map[int]*Connection{}

	loader := cfg.Loader.Load(cfg.FileName)
	ch := make(chan []sqlparser.Statement, 1000)

	_ = data.ForeachSQLQuery(loader, func(query data.Query) error {
		stmt, err := sqlparser.NewTestParser().Parse(query.Query)
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}
		switch stmt := stmt.(type) {
		case *sqlparser.Begin:
		case *sqlparser.Commit:
			connection := transactions[query.ConnectionID]
			ch <- connection.Transaction
			connection.Transaction = nil
		case *sqlparser.Set:
			for _, expr := range stmt.Exprs {
				if expr.Var.Name.Lowered() == "autocommit" {
					val, ok := expr.Expr.(*sqlparser.Literal)
					if !ok {
						continue
					}
					val2 := strings.ToLower(val.Val)
					if val2 == "1" || val2 == "on" || val2 == "true" {
						transactions[query.ConnectionID].Autocommit = true
					} else {
						transactions[query.ConnectionID].Autocommit = false
					}
				}
			}
		default:
			if !sqlparser.IsDMLStatement(stmt) {
				return nil
			}
			connection, ok := transactions[query.ConnectionID]
			if !ok {
				connection = &Connection{Autocommit: defaultAutocommit}
				transactions[query.ConnectionID] = connection
			}
			if connection.Autocommit {
				ch <- []sqlparser.Statement{stmt}
			} else {
				connection.Transaction = append(connection.Transaction, stmt)
			}
		}
		return nil
	})

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

	var txs []TxSignature
outer:
	for {
		select {
		// TODO: when a transaction has the exact same signature, increment its usage count instead of adding a new one
		case queries := <-ch:
			var tx TxSignature
			idToLiteral := make(map[string]int)
			nextID := 1
			for _, query := range queries {
				st, err := semantics.Analyze(query, "ks", si)
				if err != nil {
					panic(err)
				}

				switch query := query.(type) {
				case *sqlparser.Update:
					// Step 0:
					// We want to find all the predicates that can impact our vindex choice in the query.
					// TODO: Implement more types of predicates, right now only comparisons with 1 column and 1 literal are handled.
					// TODO: This whole step can actually be re-used for DELETE.
					//nolint:nestif // this is dirty WIP
					if query.Where != nil {
						// Step 1:
						// Find all predicates in the where clause that use a column and a literal
						var predicates []predicateInfo
						wheres := sqlparser.SplitAndExpression(nil, query.Where.Expr)
						for _, where := range wheres {
							if cmp, ok := where.(*sqlparser.ComparisonExpr); ok {
								lhs, lhsOK := cmp.Left.(*sqlparser.ColName)
								rhs, rhsOK := cmp.Right.(*sqlparser.ColName)

								if rhsStr := exprToString(cmp.Right); lhsOK && rhsStr != "" {
									predicates = append(predicates, createPredicateInfo(st, lhs, cmp.Operator, rhsStr))
								}

								if lhsStr := exprToString(cmp.Left); rhsOK && lhsStr != "" {
									switchedOp, ok := cmp.Operator.SwitchSides()
									if ok {
										predicates = append(predicates, createPredicateInfo(st, rhs, switchedOp, lhsStr))
									}
								}
							}
						}

						// Step 2:
						// Now that we have all the predicates, let's replace their literals with an ID
						for i, predicate := range predicates {
							id, ok := idToLiteral[predicate.Val]
							if !ok {
								idToLiteral[predicate.Val] = nextID
								id = nextID
								nextID++
							}
							predicates[i].Val = fmt.Sprintf(":%d", id)
							var foundOne bool
							for _, txPred := range tx.Predicates {
								if txPred == predicate {
									foundOne = true
									break
								}
							}
							if !foundOne {
								tx.Predicates = append(tx.Predicates, predicates[i])
							}
						}
					}

					// Step 3:
					// Normalize the AST our own way:
					// 	- Replace the value in SET by "v"
					// 	- Replace the literals found in where clause comparisons by the corresponding ID we got earlier
					normalizedAST := sqlparser.Rewrite(query, func(cursor *sqlparser.Cursor) bool {
						switch node := cursor.Node().(type) {
						case *sqlparser.SetExpr:
							cursor.Replace(&sqlparser.SetExpr{
								Var:  node.Var,
								Expr: sqlparser.NewArgument("v"),
							})
						case *sqlparser.Where:
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
										id, ok := idToLiteral[lhs.Val]
										if !ok {
											panic("we must be able to find a corresponding id")
										}
										newCmp.Left = sqlparser.NewArgument(strconv.Itoa(id))
										newCmp.Right = cmp.Right
									} else {
										id, ok := idToLiteral[rhs.Val]
										if !ok {
											panic("we must be able to find a corresponding id")
										}
										newCmp.Right = sqlparser.NewArgument(strconv.Itoa(id))
										newCmp.Left = cmp.Left
									}
									newWhere.Expr = sqlparser.AndExpressions(newWhere.Expr, &newCmp)
								default:
									newWhere.Expr = sqlparser.AndExpressions(newWhere.Expr, where)
								}
							}
							cursor.Replace(&newWhere)
						}
						return true
					}, nil)
					tx.Queries = append(tx.Queries, sqlparser.String(normalizedAST))
				default:
					panic("not supported for now")
				}
			}
			txs = append(txs, tx)
		default:
			break outer
		}
	}

	txsJSON, err := json.MarshalIndent(txs, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(out, "%s\n", string(txsJSON))
}

func createPredicateInfo(st *semantics.SemTable, expr *sqlparser.ColName, op sqlparser.ComparisonExprOperator, value string) predicateInfo {
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
		Val:   value,
	}
}

func exprToString(expr sqlparser.Expr) string {
	if v, ok := expr.(*sqlparser.Literal); ok {
		return v.Val
	}
	return ""
}

func GetAutocommitGuess(cfg Config) bool {
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

		stmt, err := sqlparser.NewTestParser().Parse(query.Query)
		if err != nil {
			fmt.Println(err.Error())
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
