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
	"cmp"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"

	"vitess.io/vitess/go/vt/sqlparser"
)

type (
	TxSignature struct {
		Queries    []string        `json:"queries"`
		Predicates []predicateInfo `json:"predicates"`
		Count      int             `json:"count"`
	}

	txSignatureMap struct {
		data map[uint64][]*TxSignature
	}

	predicateInfo struct {
		Table string
		Col   string
		Op    sqlparser.ComparisonExprOperator
		Val   int
	}
)

func (pi predicateInfo) String() string {
	val := strconv.Itoa(pi.Val)
	if pi.Val == -1 {
		val = "?"
	}
	return fmt.Sprintf("%s.%s %s %s", pi.Table, pi.Col, pi.Op.ToString(), val)
}

func (pi predicateInfo) compareTo(b predicateInfo) int {
	if pi.Table != b.Table {
		return cmp.Compare(pi.Table, b.Table)
	}
	if pi.Col != b.Col {
		return cmp.Compare(pi.Col, b.Col)
	}
	if pi.Op != b.Op {
		return cmp.Compare(pi.Op, b.Op)
	}
	return cmp.Compare(pi.Val, b.Val)
}

func (tx *TxSignature) MarshalJSON() ([]byte, error) {
	predicateStrings := make([]string, len(tx.Predicates))
	for i, predicate := range tx.Predicates {
		predicateStrings[i] = predicate.String()
	}

	return json.Marshal(struct {
		Queries    []string `json:"query-signatures"`
		Predicates []string `json:"predicates"`
		Count      int      `json:"count"`
	}{
		Queries:    tx.Queries,
		Predicates: predicateStrings,
		Count:      tx.Count,
	})
}

func (tx *TxSignature) Hash64() uint64 {
	hasher := fnv.New64a()

	for _, query := range tx.Queries {
		_, _ = hasher.Write([]byte(query))
		_, _ = hasher.Write([]byte{0})
	}

	for _, pred := range tx.Predicates {
		_, _ = hasher.Write([]byte(pred.String()))
		_, _ = hasher.Write([]byte{0})
	}

	return hasher.Sum64()
}

func (tx *TxSignature) addPredicate(predicates []predicateInfo) {
	for _, predicate := range predicates {
		index := sort.Search(len(tx.Predicates), func(i int) bool {
			return tx.Predicates[i].compareTo(predicate) >= 0
		})

		if index < len(tx.Predicates) && tx.Predicates[index].compareTo(predicate) == 0 {
			continue // Predicate already exists; skip it
		}

		// Insert the predicate at the correct position
		tx.Predicates = append(tx.Predicates, predicate)     // Expand the slice by one
		copy(tx.Predicates[index+1:], tx.Predicates[index:]) // Shift elements to the right
		tx.Predicates[index] = predicate                     // Place the new predicate
	}
}

func newTxSignatureMap() *txSignatureMap {
	return &txSignatureMap{
		data: make(map[uint64][]*TxSignature),
	}
}

func (m *txSignatureMap) Add(tx *TxSignature) {
	hash := tx.Hash64()

	bucket, exists := m.data[hash]

	// Check if the hash already exists
	if !exists {
		tx.Count = 1
		m.data[hash] = []*TxSignature{tx}
		return
	}

	// Iterate over the bucket to check for exact match
	for _, existingTx := range bucket {
		if tx.Equals(existingTx) {
			existingTx.Count++
			return
		}
	}

	// No exact match found; append to the bucket
	m.data[hash] = append(bucket, tx)
}

func (tx *TxSignature) Equals(other *TxSignature) bool {
	if len(tx.Queries) != len(other.Queries) {
		return false
	}
	for i := range tx.Queries {
		if tx.Queries[i] != other.Queries[i] {
			return false
		}
	}

	if len(tx.Predicates) != len(other.Predicates) {
		return false
	}
	for i := range tx.Predicates {
		if tx.Predicates[i] != other.Predicates[i] {
			return false
		}
	}

	return true
}

// CleanUp removes values that are only used once and replaces them with -1
func (tx *TxSignature) CleanUp() *TxSignature {
	newPredicates := make([]predicateInfo, 0, len(tx.Predicates))
	usedValues := make(map[int]int)

	// First let's count how many times each value is used
	for _, pred := range tx.Predicates {
		usedValues[pred.Val]++
	}

	// Now we replace values only used once with -1
	newCount := 0
	newValues := make(map[int]int)
	for _, pred := range tx.Predicates {
		if usedValues[pred.Val] == 1 {
			pred.Val = -1
		} else {
			newVal, found := newValues[pred.Val]
			if !found {
				// Assign a new value to this predicate
				newVal = newCount
				newCount++
				newValues[pred.Val] = newVal
			}
			pred.Val = newVal
		}
		newPredicates = append(newPredicates, pred)
	}

	return &TxSignature{
		Queries:    tx.Queries,
		Predicates: newPredicates,
		Count:      tx.Count,
	}
}

func (m *txSignatureMap) MarshalJSON() ([]byte, error) {
	// Collect all interesting TxSignatures into a slice
	var signatures []*TxSignature
	for _, bucket := range m.data {
		for _, txSig := range bucket {
			if txSig.Count > 1 {
				signatures = append(signatures, txSig.CleanUp())
			}
		}
	}

	sort.Slice(signatures, func(i, j int) bool {
		return signatures[i].Count > signatures[j].Count
	})

	result := map[string]any{
		"fileType":   "transactions",
		"signatures": signatures,
	}

	return json.Marshal(result)
}
