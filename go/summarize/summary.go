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
	"io"
	"strings"
	"time"

	"vitess.io/vitess/go/vt/vtgate/planbuilder/operators"

	"github.com/vitessio/vt/go/dbinfo"
	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/markdown"
)

type (
	Summary struct {
		tables        []*TableSummary
		failures      []FailuresSummary
		transactions  []TransactionSummary
		hotQueries    []keys.QueryAnalysisResult
		hotQueryFn    getMetric
		analyzedFiles []string
		queryGraph    queryGraph
		joins         []joinDetails
		hasRowCount   bool
	}

	TableSummary struct {
		Table            string
		ReadQueryCount   int
		WriteQueryCount  int
		ColumnUses       map[ColumnInformation]ColumnUsage
		JoinPredicates   []operators.JoinPredicate
		Failed           bool
		RowCount         int
		ReferencedTables []*dbinfo.ForeignKey
	}

	TransactionSummary struct {
		Count   int
		Queries []QueryPattern

		// Joins contain a list of columns that are joined together.
		// Each outer slice is one set of columns that are joined together.
		Joins [][]string
	}

	QueryPattern struct {
		Type           string
		Table          string
		Predicates     []string
		UpdatedColumns []string
	}
)

func NewSummary(hotMetric string) *Summary {
	return &Summary{
		queryGraph: make(queryGraph),
		hotQueryFn: getMetricForHotness(hotMetric),
	}
}

func (s *Summary) PrintMarkdown(out io.Writer, now time.Time) {
	md := &markdown.MarkDown{}
	filePlural := ""
	msg := `# Query Analysis Report

**Date of Analysis**: %s  
**Analyzed File%s**: ` + "%s" + `

`
	if len(s.analyzedFiles) > 1 {
		filePlural = "s"
	}
	for i, file := range s.analyzedFiles {
		s.analyzedFiles[i] = "`" + file + "`"
	}
	md.Printf(msg, now.Format(time.DateTime), filePlural, strings.Join(s.analyzedFiles, ", "))
	renderHotQueries(md, s.hotQueries, s.hotQueryFn)
	renderTableUsage(md, s.tables, s.hasRowCount)
	renderTablesJoined(md, s)
	renderTransactions(md, s.transactions)
	renderFailures(md, s.failures)

	_, err := md.WriteTo(out)
	if err != nil {
		panic(err)
	}
}

func (s *Summary) GetTable(name string) *TableSummary {
	for _, table := range s.tables {
		if table.Table == name {
			return table
		}
	}
	return nil
}

func (s *Summary) AddTable(table *TableSummary) {
	s.tables = append(s.tables, table)
}
