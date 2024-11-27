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
	"iter"
	"slices"
	"sort"

	"vitess.io/vitess/go/slice"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/planbuilder/operators"

	"github.com/vitessio/vt/go/keys"
)

const HotQueryCount = 10

type (
	Position int

	ColumnUsage struct {
		Percentage float64
		Count      int
	}

	ColumnInformation struct {
		Name string
		Pos  Position
	}

	FailuresSummary struct {
		Error string
		Count int
	}

	graphKey struct {
		Tbl1, Tbl2 string
	}

	queryGraph map[graphKey]map[operators.JoinPredicate]int
)

const (
	Join Position = iota
	JoinRange
	Where
	WhereRange
	Grouping
)

func (p Position) String() string {
	switch p {
	case Join:
		return "JOIN"
	case JoinRange:
		return "JOIN RANGE"
	case Where:
		return "WHERE"
	case WhereRange:
		return "WHERE RANGE"
	case Grouping:
		return "GROUP"
	}

	panic("unknown Position")
}

func (ci ColumnInformation) String() string {
	return fmt.Sprintf("%s %s", ci.Name, ci.Pos)
}

func (ts TableSummary) GetColumns() iter.Seq2[ColumnInformation, ColumnUsage] {
	type colDetails struct {
		ci ColumnInformation
		cu ColumnUsage
	}
	columns := make([]colDetails, 0, len(ts.ColumnUses))
	maxColUse := make(map[string]float64)
	for colInfo, usage := range ts.ColumnUses {
		columns = append(columns, colDetails{ci: colInfo, cu: usage})
		if maxColUse[colInfo.Name] < usage.Percentage {
			maxColUse[colInfo.Name] = usage.Percentage
		}
	}

	sort.Slice(columns, func(i, j int) bool {
		nameI := columns[i].ci.Name
		nameJ := columns[j].ci.Name
		maxPercenI := maxColUse[nameI]
		maxPercenJ := maxColUse[nameJ]

		if nameI == nameJ {
			if columns[i].cu.Percentage == columns[j].cu.Percentage {
				return columns[i].ci.Pos < columns[j].ci.Pos
			}
			return columns[i].cu.Percentage > columns[j].cu.Percentage
		}
		if maxPercenI == maxPercenJ {
			return nameI < nameJ
		}

		return maxPercenI > maxPercenJ
	})
	return func(yield func(ColumnInformation, ColumnUsage) bool) {
		for _, col := range columns {
			if !yield(col.ci, col.cu) {
				break
			}
		}
	}
}

func (ts TableSummary) UseCount() int {
	return ts.ReadQueryCount + ts.WriteQueryCount
}

type getMetric = func(q keys.QueryAnalysisResult) float64

func getMetricForHotness(metric string) getMetric {
	switch metric {
	case "usage-count":
		return func(q keys.QueryAnalysisResult) float64 {
			return float64(q.UsageCount)
		}
	case "total-rows-examined":
		return func(q keys.QueryAnalysisResult) float64 {
			return float64(q.RowsExamined)
		}
	case "avg-rows-examined":
		return func(q keys.QueryAnalysisResult) float64 {
			return float64(q.RowsExamined) / float64(q.UsageCount)
		}
	case "total-time", "":
		return func(q keys.QueryAnalysisResult) float64 {
			return q.QueryTime
		}
	case "avg-time":
		return func(q keys.QueryAnalysisResult) float64 {
			return q.QueryTime / float64(q.UsageCount)
		}
	default:
		exit(fmt.Sprintf("unknown metric: %s", metric))
		panic("unreachable")
	}
}

func (g queryGraph) AddJoinPredicate(key graphKey, pred operators.JoinPredicate) {
	if in, exists := g[key]; exists {
		in[pred]++
		return
	}

	g[key] = map[operators.JoinPredicate]int{pred: 1}
}

// makeKey creates a graph key from two columns. The key is always sorted in ascending order.
func makeKey(lhs, rhs operators.Column) graphKey {
	if lhs.Table < rhs.Table {
		return graphKey{lhs.Table, rhs.Table}
	}

	return graphKey{rhs.Table, lhs.Table}
}

func summarizeKeysQueries(summary *Summary, queries *keys.Output) {
	tableSummaries := make(map[string]*TableSummary)
	tableUsageWriteCounts := make(map[string]int)
	tableUsageReadCounts := make(map[string]int)

	// First pass: collect all data and count occurrences
	for _, query := range queries.Queries {
		gatherTableInfo(query, tableSummaries, tableUsageWriteCounts, tableUsageReadCounts)
		checkQueryForHotness(&summary.hotQueries, query, summary.hotQueryFn)
	}

	// Second pass: calculate percentages
	for _, tblSummary := range tableSummaries {
		tblSummary.ReadQueryCount = tableUsageReadCounts[tblSummary.Table]
		tblSummary.WriteQueryCount = tableUsageWriteCounts[tblSummary.Table]
		count := tblSummary.ReadQueryCount + tblSummary.WriteQueryCount
		countF := float64(count)
		for colName, usage := range tblSummary.ColumnUses {
			usage.Percentage = (float64(usage.Count) / countF) * 100
			tblSummary.ColumnUses[colName] = usage
		}
	}

	// Convert map to slice
	for _, tblSummary := range tableSummaries {
		table := summary.GetTable(tblSummary.Table)
		if table == nil {
			summary.AddTable(tblSummary)
			continue
		}
		table.ReadQueryCount = tblSummary.ReadQueryCount
		table.WriteQueryCount = tblSummary.WriteQueryCount
		if table.ColumnUses != nil {
			panic("ColumnUses already set for table" + tblSummary.Table)
		}
		table.ColumnUses = tblSummary.ColumnUses
		if table.JoinPredicates != nil {
			panic("JoinPredicates already set for table" + tblSummary.Table)
		}
		table.JoinPredicates = tblSummary.JoinPredicates
	}

	// Collect failed queries
	var failures []FailuresSummary
	for _, query := range queries.Failed {
		failures = append(failures, FailuresSummary{
			Error: query.Error,
			Count: len(query.LineNumbers),
		})
	}
	summary.failures = failures

	for _, query := range queries.Queries {
		for _, pred := range query.JoinPredicates {
			key := makeKey(pred.LHS, pred.RHS)
			summary.queryGraph.AddJoinPredicate(key, pred)
		}
	}
}

func checkQueryForHotness(hotQueries *[]keys.QueryAnalysisResult, query keys.QueryAnalysisResult, metricReader getMetric) {
	// todo: we should be able to choose different metrics for hotness - e.g. total time spent on query, number of rows examined, etc.
	switch {
	case len(*hotQueries) < HotQueryCount:
		// If we have not yet reached the limit, add the query
		*hotQueries = append(*hotQueries, query)
	case metricReader(query) > metricReader((*hotQueries)[0]):
		// If the current query has more usage than the least used hot query, replace it
		(*hotQueries)[0] = query
	default:
		// If the current query is not hot enough, just return
		return
	}

	// Sort the hot queries by query time so that the least used query is always at the front
	sort.Slice(*hotQueries,
		func(i, j int) bool {
			return metricReader((*hotQueries)[i]) < metricReader((*hotQueries)[j])
		})
}

func gatherTableInfo(query keys.QueryAnalysisResult, tableSummaries map[string]*TableSummary, tableUsageWriteCounts map[string]int, tableUsageReadCounts map[string]int) {
	for _, table := range query.TableNames {
		if _, exists := tableSummaries[table]; !exists {
			tableSummaries[table] = &TableSummary{
				Table:      table,
				ColumnUses: make(map[ColumnInformation]ColumnUsage),
			}
		}

		switch query.StatementType {
		case "INSERT", "DELETE", "UPDATE", "REPLACE":
			tableUsageWriteCounts[table] += query.UsageCount
		default:
			tableUsageReadCounts[table] += query.UsageCount
		}

		summarizeColumnUsage(tableSummaries[table], query)
		summarizeJoinPredicates(query.JoinPredicates, table, tableSummaries)
	}
}

func summarizeColumnUsage(tableSummary *TableSummary, query keys.QueryAnalysisResult) {
	updateColumnUsage := func(columns []ColumnInformation) {
		sort.Slice(columns, func(i, j int) bool {
			if columns[i].Name == columns[j].Name {
				return columns[i].Pos < columns[j].Pos
			}
			return columns[i].Name < columns[j].Name
		})
		columns = slices.Compact(columns)

		for _, col := range columns {
			usage := tableSummary.ColumnUses[col]
			usage.Count += query.UsageCount
			tableSummary.ColumnUses[col] = usage
		}
	}

	updateColumnUsage(slice.Map(slice.Filter(query.FilterColumns, func(col operators.ColumnUse) bool {
		return col.Column.Table == tableSummary.Table
	}), func(col operators.ColumnUse) ColumnInformation {
		pos := Where
		if col.Uses != sqlparser.EqualOp {
			pos = WhereRange
		}
		return ColumnInformation{Name: col.Column.Name, Pos: pos}
	}))

	updateColumnUsage(slice.Map(slice.Filter(query.GroupingColumns, func(col operators.Column) bool {
		return col.Table == tableSummary.Table
	}), func(col operators.Column) ColumnInformation {
		return ColumnInformation{Name: col.Name, Pos: Grouping}
	}))

	updateColumnUsage(slice.Map(slice.Filter(query.JoinPredicates, func(pred operators.JoinPredicate) bool {
		return pred.LHS.Table == tableSummary.Table || pred.RHS.Table == tableSummary.Table
	}), func(pred operators.JoinPredicate) ColumnInformation {
		ci := ColumnInformation{Pos: Join}
		if pred.Uses != sqlparser.EqualOp {
			ci.Pos = JoinRange
		}
		switch tableSummary.Table {
		case pred.LHS.Table:
			ci.Name = pred.LHS.Name
		case pred.RHS.Table:
			ci.Name = pred.RHS.Name
		}
		return ci
	}))
}

func summarizeJoinPredicates(joinPredicates []operators.JoinPredicate, table string, tableSummaries map[string]*TableSummary) {
outer:
	for _, predicate := range joinPredicates {
		if predicate.LHS.Table != table && predicate.RHS.Table != table {
			// should never be true, but just in case something went wrong
			continue
		}
		for _, joinPredicate := range tableSummaries[table].JoinPredicates {
			if joinPredicate == predicate {
				continue outer
			}
		}
		tableSummaries[table].JoinPredicates = append(tableSummaries[table].JoinPredicates, predicate)
	}
}
