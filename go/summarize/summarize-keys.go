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
	"strings"
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

	TableSummary struct {
		Table           string
		ReadQueryCount  int
		WriteQueryCount int
		Columns         map[ColumnInformation]ColumnUsage
		JoinPredicates  []operators.JoinPredicate
		Failed          bool
	}

	FailuresSummary struct {
		Query string
		Error string
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
	md.Printf(msg, file.Name, now.Format(time.DateTime))

	tableSummaries, failuresSummaries := summarizeKeysQueries(file.AnalysedQueries)

	renderTableUsage(tableSummaries, md)
	renderTablesJoined(md, file.AnalysedQueries)
	renderFailures(md, failuresSummaries)

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
	for colInfo, usage := range summary.GetColumns() {
		rows = append(rows, []string{
			colInfo.Name,
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
	ks := slices.Collect(maps.Keys(g))
	slices.SortFunc(ks, func(a, b graphKey) int {
		if a.Tbl1 == b.Tbl1 {
			return strings.Compare(a.Tbl2, b.Tbl2)
		}
		return strings.Compare(a.Tbl1, b.Tbl1)
	})

	if len(g) > 0 {
		md.PrintHeader("Tables Joined", 2)
	}

	// we really want the output to be deterministic
	tables := slices.Collect(maps.Keys(g))
	sort.Slice(tables, func(i, j int) bool {
		if tables[i].Tbl1 == tables[j].Tbl1 {
			return tables[i].Tbl2 < tables[j].Tbl2
		}
		return tables[i].Tbl1 < tables[j].Tbl1
	})

	for _, table := range tables {
		predicates := g[table]
		md.Println("```")
		md.Printf("%s ↔ %s\n", table.Tbl1, table.Tbl2)
		numberOfPreds := len(predicates)
		totalt := 0
		for _, count := range predicates {
			totalt += count
		}

		for predicate, count := range predicates {
			numberOfPreds--
			var s string
			if numberOfPreds == 0 {
				s = "└─"
			} else {
				s = "├─"
			}
			md.Printf("%s %s %d%%\n", s, predicate.String(), (count*100)/totalt)
		}
		md.Println("```")
		md.NewLine()
	}
}

func renderFailures(md *markdown.MarkDown, failures []FailuresSummary) {
	if len(failures) == 0 {
		return
	}
	md.PrintHeader("Failures", 2)

	headers := []string{"Query", "Error"}
	var rows [][]string
	for _, failure := range failures {
		rows = append(rows, []string{failure.Query, failure.Error})
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

func summarizeKeysQueries(queries *keys.Output) ([]TableSummary, []FailuresSummary) {
	tableSummaries := make(map[string]*TableSummary)
	tableUsageWriteCounts := make(map[string]int)
	tableUsageReadCounts := make(map[string]int)

	// First pass: collect all data and count occurrences
	for _, query := range queries.Queries {
		for _, table := range query.TableName {
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

			summarizeColumnUsage(table, tableSummaries, query)
			summarizeJoinPredicates(query.JoinPredicates, table, tableSummaries)
		}
	}

	// Second pass: calculate percentages
	for _, summary := range tableSummaries {
		summary.ReadQueryCount = tableUsageReadCounts[summary.Table]
		summary.WriteQueryCount = tableUsageWriteCounts[summary.Table]
		count := summary.ReadQueryCount + summary.WriteQueryCount
		for colName, usage := range summary.Columns {
			countF := float64(count)
			usage.Percentage = (usage.Percentage / countF) * 100
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
			Query: query.Query,
			Error: query.Error,
		})
	}

	return result, failures
}

func summarizeColumnUsage(table string, tableSummaries map[string]*TableSummary, query keys.QueryAnalysisResult) {
	updateColumnUsage := func(columns []ColumnInformation) {
		sort.Slice(columns, func(i, j int) bool {
			if columns[i].Name == columns[j].Name {
				return columns[i].Pos < columns[j].Pos
			}
			return columns[i].Name < columns[j].Name
		})
		columns = slices.Compact(columns)

		for _, col := range columns {
			if strings.HasPrefix(col.Name, table+".") {
				col.Name = strings.TrimPrefix(col.Name, table+".")
				usage := tableSummaries[table].Columns[col]
				queryUsageCount := query.UsageCount
				usage.Percentage += float64(queryUsageCount)
				usage.Count += queryUsageCount
				tableSummaries[table].Columns[col] = usage
			}
		}
	}

	updateColumnUsage(slice.Map(query.FilterColumns, func(col operators.ColumnUse) ColumnInformation {
		pos := Where
		if col.Uses != sqlparser.EqualOp {
			pos = WhereRange
		}
		return ColumnInformation{Name: col.Column.String(), Pos: pos}
	}))

	updateColumnUsage(slice.Map(query.GroupingColumns, func(col operators.Column) ColumnInformation {
		return ColumnInformation{Name: col.String(), Pos: Grouping}
	}))

	updateColumnUsage(slice.Map(query.JoinPredicates, func(pred operators.JoinPredicate) ColumnInformation {
		ci := ColumnInformation{Pos: Join}
		if pred.Uses != sqlparser.EqualOp {
			ci.Pos = JoinRange
		}
		switch table {
		case pred.LHS.Table:
			ci.Name = pred.LHS.String()
		case pred.RHS.Table:
			ci.Name = pred.RHS.String()
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
