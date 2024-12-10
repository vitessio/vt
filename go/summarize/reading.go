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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/vitessio/vt/go/dbinfo"
	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/planalyze"
	"github.com/vitessio/vt/go/transactions"
)

func readTracedFile(fileName string) (traceSummary, error) {
	c, err := os.ReadFile(fileName)
	if err != nil {
		return traceSummary{}, fmt.Errorf("error opening file: %w", err)
	}

	type traceOutput struct {
		FileType string        `json:"fileType"`
		Queries  []TracedQuery `json:"queries"`
	}
	var to traceOutput
	err = json.Unmarshal(c, &to)
	if err != nil {
		return traceSummary{}, fmt.Errorf("error parsing json: %w", err)
	}

	sort.Slice(to.Queries, func(i, j int) bool {
		a, err := strconv.Atoi(to.Queries[i].LineNumber)
		if err != nil {
			return false
		}
		b, err := strconv.Atoi(to.Queries[j].LineNumber)
		if err != nil {
			return false
		}
		return a < b
	})

	return traceSummary{
		Name:          fileName,
		TracedQueries: to.Queries,
	}, nil
}

type summarizer = func(s *Summary) error

func readTransactionFile(fileName string) (summarizer, error) {
	c, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	type txOutput struct {
		FileType   string                   `json:"fileType"`
		Signatures []transactions.Signature `json:"signatures"`
	}

	var to txOutput
	err = json.Unmarshal(c, &to)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %w", err)
	}
	return func(s *Summary) error {
		s.AnalyzedFiles = append(s.AnalyzedFiles, fileName)
		return summarizeTransactions(s, to.Signatures)
	}, nil
}

func readKeysFile(fileName string) (summarizer, error) {
	ko, err := keys.ReadKeysFile(fileName)
	if err != nil {
		return nil, err
	}

	return func(s *Summary) error {
		s.AnalyzedFiles = append(s.AnalyzedFiles, fileName)
		return summarizeKeysQueries(s, &ko)
	}, nil
}

func readDBInfoFile(fileName string) (summarizer, error) {
	schemaInfo, err := dbinfo.Load(fileName)
	if err != nil {
		return nil, fmt.Errorf("error parsing dbinfo: %w", err)
	}

	return func(s *Summary) error {
		s.AnalyzedFiles = append(s.AnalyzedFiles, fileName)
		s.HasRowCount = true
		for _, ti := range schemaInfo.Tables {
			table := s.GetTable(ti.Name)
			if table == nil {
				table = &TableSummary{Table: ti.Name}
				s.AddTable(table)
			}
			table.RowCount = ti.Rows
			table.ReferencedTables = ti.ForeignKeys
		}
		return nil
	}, nil
}

func readPlanalyzeFile(filename string) (summarizer, error) {
	p, err := planalyze.ReadPlanalyzeFile(filename)
	if err != nil {
		return nil, err
	}

	return func(s *Summary) error {
		s.AnalyzedFiles = append(s.AnalyzedFiles, filename)
		return summarizePlanAnalyze(s, p)
	}, nil
}
