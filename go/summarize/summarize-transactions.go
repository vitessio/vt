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

package summarize

import (
	"fmt"
	"maps"
	"slices"

	"vitess.io/vitess/go/slice"

	"github.com/vitessio/vt/go/transactions"
)

func summarizeTransactions(s *Summary, txs []transactions.Signature) error {
	for _, tx := range txs {
		patterns, joins := summarizeQueries(tx.Queries)
		if len(joins) == 0 {
			continue
		}

		for _, p := range patterns {
			table := s.GetTable(p.Table)
			if table == nil {
				s.AddTable(&TableSummary{Table: p.Table})
			}
		}

		s.Transactions = append(s.Transactions, TransactionSummary{
			Count:   tx.Count,
			Queries: patterns,
			Joins:   joins,
		})
	}
	return nil
}

func summarizeQueries(queries []transactions.Query) (patterns []QueryPattern, joins [][]string) {
	columnJoins := map[int][]string{}
	for _, q := range queries {
		for _, predicate := range q.Predicates {
			if predicate.Val >= 0 {
				columnJoins[predicate.Val] = append(columnJoins[predicate.Val], fmt.Sprintf("%s.%s", q.AffectedTable, predicate.Col))
			}
		}
		patterns = append(patterns, QueryPattern{
			Type:           q.Op,
			Table:          q.AffectedTable,
			Predicates:     slice.Map(q.Predicates, func(p transactions.PredicateInfo) string { return p.String() }),
			UpdatedColumns: q.UpdatedColumns,
		})
	}
	joinKeys := slices.Collect(maps.Keys(columnJoins))
	slices.Sort(joinKeys)

	for _, key := range joinKeys {
		cols := columnJoins[key]
		joins = append(joins, cols)
	}
	return
}
