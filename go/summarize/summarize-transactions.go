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
	"vitess.io/vitess/go/slice"

	"github.com/vitessio/vt/go/transactions"
)

func summarizeTransactions(s *Summary, txs []transactions.Signature) error {
	for _, tx := range txs {
		patterns, interesting := summarizeQueries(tx.Queries)
		if !interesting {
			continue
		}
		s.transactions = append(s.transactions, TransactionSummary{
			Count:   tx.Count,
			Queries: patterns,
		})
	}
	return nil
}

func summarizeQueries(queries []transactions.Query) (patterns []QueryPattern, interesting bool) {
	for _, q := range queries {
		if !interesting {
			// Check if any of the predicates are interesting
			for _, predicate := range q.Predicates {
				if predicate.Val >= 0 {
					interesting = true
				}
			}
		}
		patterns = append(patterns, QueryPattern{
			Type:           q.Op,
			Table:          q.AffectedTable,
			Predicates:     slice.Map(q.Predicates, func(p transactions.PredicateInfo) string { return p.String() }),
			UpdatedColumns: q.UpdatedColumns,
		})
	}
	return
}
