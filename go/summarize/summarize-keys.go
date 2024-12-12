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
	"maps"
	"slices"
	"sort"
	"strings"

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

	joinDetails struct {
		Tbl1, Tbl2  string
		Occurrences int
		Predicates  []operators.JoinPredicate
	}

	queryGraph map[graphKey]map[operators.JoinPredicate]int
)

func (ci *ColumnInformation) String() string {
	return fmt.Sprintf("%s/%s", ci.Name, ci.Pos)
}

func ColumnInfoFromString(s string) (*ColumnInformation, error) {
	ci := ColumnInformation{}
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid column information: %s", s)
	}
	ci.Name = parts[0]
	pos, err := PositionFromString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid column position: %s", parts[1])
	}
	ci.Pos = pos
	return &ci, nil
}

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

	return "UNKNOWN"
}

func PositionFromString(s string) (Position, error) {
	switch s {
	case "JOIN":
		return Join, nil
	case "JOIN RANGE":
		return JoinRange, nil
	case "WHERE":
		return Where, nil
	case "WHERE RANGE":
		return WhereRange, nil
	case "GROUP":
		return Grouping, nil
	}

	return 0, fmt.Errorf("invalid position: %s", s)
}

type TemplateColumn struct {
	ColInfo ColumnInformation
	Usage   ColumnUsage
}

func (ts TableSummary) GetColumnsSlice() []TemplateColumn {
	columns := make([]TemplateColumn, 0, len(ts.ColumnUses))
	for colInfoKey, usage := range ts.ColumnUses {
		colInfo, err := ColumnInfoFromString(colInfoKey)
		if err != nil {
			panic(err)
		}
		columns = append(columns, TemplateColumn{ColInfo: *colInfo, Usage: usage})
	}
	return columns
}

func (ts TableSummary) GetColumns() iter.Seq2[ColumnInformation, ColumnUsage] {
	type colDetails struct {
		ci ColumnInformation
		cu ColumnUsage
	}
	columns := make([]colDetails, 0, len(ts.ColumnUses))
	maxColUse := make(map[string]float64)
	for colInfoKey, usage := range ts.ColumnUses {
		colInfo, err := ColumnInfoFromString(colInfoKey)
		if err != nil {
			panic(err)
		}
		columns = append(columns, colDetails{ci: *colInfo, cu: usage})
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

func getMetricForHotness(metric string) (getMetric, error) {
	switch metric {
	case "usage-count":
		return func(q keys.QueryAnalysisResult) float64 {
			return float64(q.UsageCount)
		}, nil
	case "total-rows-examined":
		return func(q keys.QueryAnalysisResult) float64 {
			return float64(q.RowsExamined)
		}, nil
	case "avg-rows-examined":
		return func(q keys.QueryAnalysisResult) float64 {
			return float64(q.RowsExamined) / float64(q.UsageCount)
		}, nil
	case "total-time", "":
		return func(q keys.QueryAnalysisResult) float64 {
			return q.QueryTime
		}, nil
	case "avg-time":
		return func(q keys.QueryAnalysisResult) float64 {
			return q.QueryTime / float64(q.UsageCount)
		}, nil
	default:
		return nil, fmt.Errorf("unknown metric: %s", metric)
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
	lhsTable := strings.Trim(lhs.Table, "`")
	rhsTable := strings.Trim(rhs.Table, "`")
	if lhsTable < rhsTable {
		return graphKey{lhsTable, rhsTable}
	}

	return graphKey{rhsTable, lhsTable}
}

func summarizeKeysQueries(summary *Summary, queries *keys.Output) error {
	tableSummaries := make(map[string]*TableSummary)
	tableUsageWriteCounts := make(map[string]int)
	tableUsageReadCounts := make(map[string]int)

	// First pass: collect all graphData and count occurrences
	for _, query := range queries.Queries {
		summary.addQueryResult(query)
		gatherTableInfo(query, tableSummaries, tableUsageWriteCounts, tableUsageReadCounts)
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
			return fmt.Errorf("ColumnUses already set for table %s", tblSummary.Table)
		}
		table.ColumnUses = tblSummary.ColumnUses
		if table.JoinPredicates != nil {
			return fmt.Errorf("JoinPredicates already set for table %s", tblSummary.Table)
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
	summary.Failures = failures

	for _, query := range queries.Queries {
		for _, pred := range query.JoinPredicates {
			key := makeKey(pred.LHS, pred.RHS)
			summary.queryGraph.AddJoinPredicate(key, pred)
		}
	}

	for tables, predicates := range summary.queryGraph {
		occurrences := 0
		for _, count := range predicates {
			occurrences += count
		}
		joinPredicates := slices.Collect(maps.Keys(predicates))
		sort.Slice(joinPredicates, func(i, j int) bool {
			return joinPredicates[i].String() < joinPredicates[j].String()
		})
		summary.Joins = append(summary.Joins, joinDetails{
			Tbl1:        tables.Tbl1,
			Tbl2:        tables.Tbl2,
			Occurrences: occurrences,
			Predicates:  joinPredicates,
		})
	}
	sort.Slice(summary.Joins, func(i, j int) bool {
		if summary.Joins[i].Occurrences != summary.Joins[j].Occurrences {
			return summary.Joins[i].Occurrences > summary.Joins[j].Occurrences
		}
		if summary.Joins[i].Tbl1 != summary.Joins[j].Tbl1 {
			return summary.Joins[i].Tbl1 < summary.Joins[j].Tbl1
		}
		return summary.Joins[i].Tbl2 < summary.Joins[j].Tbl2
	})
	return nil
}

func checkQueryForHotness(hotQueries *[]HotQueryResult, query QueryResult, metricReader getMetric) {
	// todo: we should be able to choose different metrics for hotness - e.g. total time spent on query, number of rows examined, etc.
	newHotQueryFn := func() HotQueryResult {
		return HotQueryResult{
			QueryResult: QueryResult{
				QueryAnalysisResult: query.QueryAnalysisResult,
				PlanAnalysis:        query.PlanAnalysis,
			},
			AvgQueryTime: query.QueryAnalysisResult.QueryTime / float64(query.QueryAnalysisResult.UsageCount),
		}
	}

	switch {
	case len(*hotQueries) < HotQueryCount:
		// If we have not yet reached the limit, add the query
		*hotQueries = append(*hotQueries, newHotQueryFn())
	case metricReader(query.QueryAnalysisResult) > metricReader((*hotQueries)[0].QueryAnalysisResult):
		// If the current query has more usage than the least used hot query, replace it
		(*hotQueries)[0] = newHotQueryFn()
	default:
		// If the current query is not hot enough, just return
		return
	}

	// Sort the hot queries by query time so that the least used query is always at the front
	sort.Slice(*hotQueries,
		func(i, j int) bool {
			return metricReader((*hotQueries)[i].QueryAnalysisResult) < metricReader((*hotQueries)[j].QueryAnalysisResult)
		})
}

func gatherTableInfo(query keys.QueryAnalysisResult, tableSummaries map[string]*TableSummary, tableUsageWriteCounts map[string]int, tableUsageReadCounts map[string]int) {
	for _, table := range query.TableNames {
		if _, exists := tableSummaries[table]; !exists {
			tableSummaries[table] = &TableSummary{
				Table:      table,
				ColumnUses: make(map[string]ColumnUsage),
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
			usage := tableSummary.ColumnUses[col.String()]
			usage.Count += query.UsageCount
			tableSummary.ColumnUses[col.String()] = usage
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
