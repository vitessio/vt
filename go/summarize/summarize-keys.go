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
	"io"
	"iter"
	"maps"
	"slices"
	"sort"
	"strconv"
	"time"

	"vitess.io/vitess/go/slice"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/planbuilder/operators"

	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/markdown"
)

type (
	Position    int
	ColumnUsage struct {
		Percentage float64
		Count      int
	}
	ColumnInformation struct {
		Name string
		Pos  Position
	}

	Summary struct {
		tables   []TableSummary
		failures []FailuresSummary
	}

	TableSummary struct {
		Table           string
		ReadQueryCount  int
		WriteQueryCount int
		Columns         map[ColumnInformation]ColumnUsage
		JoinPredicates  []operators.JoinPredicate
		Failed          bool
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
	columns := make([]colDetails, 0, len(ts.Columns))
	maxColUse := make(map[string]float64)
	for colInfo, usage := range ts.Columns {
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

// printKeysSummary goes over all the analysed queries, gathers information about column usage per table,
// and prints this summary information to the output.
func printKeysSummary(out io.Writer, file readingSummary, now time.Time) {
	md := &markdown.MarkDown{}

	msg := `# Query Analysis Report

**Date of Analysis**: %s  
**Analyzed File**: ` + "`%s`" + `

`
	md.Printf(msg, now.Format(time.DateTime), file.Name)

	summary := summarizeKeysQueries(file.AnalysedQueries)

	renderTableUsage(summary.tables, md)
	renderTablesJoined(md, file.AnalysedQueries)
	renderFailures(md, summary.failures)

	_, err := md.WriteTo(out)
	if err != nil {
		panic(err)
	}
}

func renderTableUsage(tableSummaries []TableSummary, md *markdown.MarkDown) {
	if len(tableSummaries) == 0 {
		return
	}

	sort.Slice(tableSummaries, func(i, j int) bool {
		return tableSummaries[i].UseCount() > tableSummaries[j].UseCount()
	})

	md.PrintHeader("Tables", 2)
	renderTableOverview(md, tableSummaries)

	md.PrintHeader("Column Usage", 3)
	for _, summary := range tableSummaries {
		renderColumnUsageTable(md, summary)
	}
}

func renderTableOverview(md *markdown.MarkDown, tableSummaries []TableSummary) {
	headers := []string{"Table Name", "Reads", "Writes"}
	var rows [][]string
	for _, summary := range tableSummaries {
		rows = append(rows, []string{
			summary.Table,
			strconv.Itoa(summary.ReadQueryCount),
			strconv.Itoa(summary.WriteQueryCount),
		})
	}
	md.PrintTable(headers, rows)
}

func renderColumnUsageTable(md *markdown.MarkDown, summary TableSummary) {
	md.PrintHeader(fmt.Sprintf("Table: `%s` (%d reads and %d writes)", summary.Table, summary.ReadQueryCount, summary.WriteQueryCount), 4)

	headers := []string{"Column", "Position", "Used %"}
	var rows [][]string
	var lastName string
	for colInfo, usage := range summary.GetColumns() {
		name := colInfo.Name
		if lastName == name {
			name = ""
		} else {
			lastName = name
		}
		rows = append(rows, []string{
			name,
			colInfo.Pos.String(),
			fmt.Sprintf("%.0f%%", usage.Percentage),
		})
	}

	md.PrintTable(headers, rows)
}

func (g queryGraph) AddJoinPredicate(key graphKey, pred operators.JoinPredicate) {
	if in, exists := g[key]; exists {
		in[pred]++
		return
	}

	g[key] = map[operators.JoinPredicate]int{pred: 1}
}

func renderTablesJoined(md *markdown.MarkDown, summary *keys.Output) {
	g := make(queryGraph)
	for _, query := range summary.Queries {
		for _, pred := range query.JoinPredicates {
			key := makeKey(pred.LHS, pred.RHS)
			g.AddJoinPredicate(key, pred)
		}
	}

	if len(g) > 0 {
		md.PrintHeader("Tables Joined", 2)
	}

	type joinDetails struct {
		Tbl1, Tbl2  string
		Occurrences int
		predicates  []operators.JoinPredicate
	}

	var joins []joinDetails
	for tables, predicates := range g {
		occurrences := 0
		for _, count := range predicates {
			occurrences += count
		}
		joinPredicates := slices.Collect(maps.Keys(predicates))
		sort.Slice(joinPredicates, func(i, j int) bool {
			return joinPredicates[i].String() < joinPredicates[j].String()
		})
		joins = append(joins, joinDetails{
			Tbl1:        tables.Tbl1,
			Tbl2:        tables.Tbl2,
			Occurrences: occurrences,
			predicates:  joinPredicates,
		})
	}

	sort.Slice(joins, func(i, j int) bool {
		if joins[i].Occurrences != joins[j].Occurrences {
			return joins[i].Occurrences > joins[j].Occurrences
		}
		if joins[i].Tbl1 != joins[j].Tbl1 {
			return joins[i].Tbl1 < joins[j].Tbl1
		}
		return joins[i].Tbl2 < joins[j].Tbl2
	})

	md.Println("```")
	for _, join := range joins {
		md.Printf("%s ↔ %s (Occurrences: %d)\n", join.Tbl1, join.Tbl2, join.Occurrences)
		for i, pred := range join.predicates {
			var s string
			if i == len(join.predicates)-1 {
				s = "└─"
			} else {
				s = "├─"
			}
			md.Printf("%s %s\n", s, pred.String())
		}
		md.NewLine()
	}
	md.Println("```")
}

func renderFailures(md *markdown.MarkDown, failures []FailuresSummary) {
	if len(failures) == 0 {
		return
	}
	md.PrintHeader("Failures", 2)

	headers := []string{"Error", "Count"}
	var rows [][]string
	for _, failure := range failures {
		rows = append(rows, []string{failure.Error, strconv.Itoa(failure.Count)})
	}
	md.PrintTable(headers, rows)
}

// makeKey creates a graph key from two columns. The key is always sorted in ascending order.
func makeKey(lhs, rhs operators.Column) graphKey {
	if lhs.Table < rhs.Table {
		return graphKey{lhs.Table, rhs.Table}
	}

	return graphKey{rhs.Table, lhs.Table}
}

func summarizeKeysQueries(queries *keys.Output) Summary {
	tableSummaries := make(map[string]*TableSummary)
	tableUsageWriteCounts := make(map[string]int)
	tableUsageReadCounts := make(map[string]int)

	// First pass: collect all data and count occurrences
	for _, query := range queries.Queries {
		for _, table := range query.TableNames {
			if _, exists := tableSummaries[table]; !exists {
				tableSummaries[table] = &TableSummary{
					Table:   table,
					Columns: make(map[ColumnInformation]ColumnUsage),
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

	// Second pass: calculate percentages
	for _, summary := range tableSummaries {
		summary.ReadQueryCount = tableUsageReadCounts[summary.Table]
		summary.WriteQueryCount = tableUsageWriteCounts[summary.Table]
		count := summary.ReadQueryCount + summary.WriteQueryCount
		countF := float64(count)
		for colName, usage := range summary.Columns {
			usage.Percentage = (float64(usage.Count) / countF) * 100
			summary.Columns[colName] = usage
		}
	}

	// Convert map to slice
	result := make([]TableSummary, 0, len(tableSummaries))
	for _, summary := range tableSummaries {
		result = append(result, *summary)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Table < result[j].Table
	})

	// Collect failed queries
	var failures []FailuresSummary
	for _, query := range queries.Failed {
		failures = append(failures, FailuresSummary{
			Error: query.Error,
			Count: len(query.LineNumbers),
		})
	}

	return Summary{tables: result, failures: failures}
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
			usage := tableSummary.Columns[col]
			usage.Count += query.UsageCount
			tableSummary.Columns[col] = usage
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
