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
	"testing"

	"github.com/stretchr/testify/assert"
	"vitess.io/vitess/go/vt/sqlparser"
)

func TestTxSignature_addPredicate(t *testing.T) {
	tests := []struct {
		name           string
		existing       []predicateInfo
		newOnes        []predicateInfo
		expectedResult []predicateInfo
	}{
		{
			name:     "Add single predicate to empty list",
			existing: []predicateInfo{},
			newOnes: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
			},
			expectedResult: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
			},
		},
		{
			name: "Add one predicates, have one",
			existing: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
			},
			newOnes: []predicateInfo{
				{Table: "table", Col: "name", Op: sqlparser.LikeOp, Val: 2},
			},
			expectedResult: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
				{Table: "table", Col: "name", Op: sqlparser.LikeOp, Val: 2},
			},
		},
		{
			name: "Add one predicates, have one, reverse order",
			existing: []predicateInfo{
				{Table: "table", Col: "name", Op: sqlparser.LikeOp, Val: 2},
			},
			newOnes: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
			},
			expectedResult: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
				{Table: "table", Col: "name", Op: sqlparser.LikeOp, Val: 2},
			},
		},
		{
			name: "Add existing predicate",
			existing: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
			},
			newOnes: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
			},
			expectedResult: []predicateInfo{
				{Table: "table", Col: "id", Op: sqlparser.EqualOp, Val: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &TxSignature{
				Predicates: tt.existing,
			}
			tx.addPredicate(tt.newOnes)
			assert.Equal(t, tt.expectedResult, tx.Predicates)
		})
	}
}
