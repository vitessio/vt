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
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/vitessio/vt/go/markdown"
	"vitess.io/vitess/go/slice"
	"vitess.io/vitess/go/vt/vtgate/planbuilder/operators"

	"github.com/vitessio/vt/go/keys"
)

type (
	ColumnUsage struct {
		FilterPercentage   float64
		GroupingPercentage float64
		JoinPercentage     float64
	}

	TableSummary struct {
		Table          string
		QueryCount     int
		Columns        map[string]ColumnUsage
		JoinPredicates []operators.JoinPredicate
		Failed         bool
	}

	FailuresSummary struct {
		Query string
		Error string
	}

	graphKey struct {
		Tbl1, Tbl2 string
	}

	queryGraph map[graphKey][]operators.JoinPredicate
)

func (ts TableSummary) GetColumns() iter.Seq2[string, ColumnUsage] {
	columns := make([][2]interface{}, 0, len(ts.Columns))
	for colName, usage := range ts.Columns {
		columns = append(columns, [2]interface{}{colName, usage})
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i][0].(string) < columns[j][0].(string)
	})
	return func(yield func(string, ColumnUsage) bool) {
		for _, col := range columns {
			if !yield(col[0].(string), col[1].(ColumnUsage)) {
				break
			}
		}
	}
}

// printKeysSummary goes over all the analysed queries, gathers information about column usage per table,
// and prints this summary information to the output.
func printKeysSummary(out io.Writer, file readingSummary) {
	var md = &markdown.MarkDown{}

	md.PrintHeader(fmt.Sprintf("Keys analyzed %s", time.Now().Format(time.DateTime)), 1)
	md.Println(fmt.Sprintf("File analyzed: %s", file.Name))
	md.NewLine()

	tableSummaries, _ := summarizeKeysQueries(file.AnalysedQueries)

	renderTableUsage(tableSummaries, md)
	renderJoinPredicatesUsage(md, file.AnalysedQueries)

	// if len(failuresSummaries) > 0 {
	// 	table := tablewriter.NewWriter(out)
	// 	table.SetAutoFormatHeaders(false)
	// 	table.SetHeader([]string{"Query", "Error"})
	// 	for _, summary := range failuresSummaries {
	// 		table.Append([]string{summary.Query, summary.Error})
	// 	}
	// 	fmt.Fprintf(out, "The %d following queries have failed:\n", len(failuresSummaries))
	// 	table.Render()
	// 	_, _ = fmt.Fprintln(out)
	// }
	_, err := md.WriteTo(out)
	if err != nil {
		panic(err)
	}
}

func renderTableUsage(tableSummaries []TableSummary, md *markdown.MarkDown) {
	if len(tableSummaries) == 0 {
		return
	}

	md.PrintHeader("Tables", 2)
	// TODO: high-level overview of all tables

	md.PrintHeader("Column Usage", 3)
	for _, summary := range tableSummaries {
		renderColumnUsageTable(md, summary)
	}
}

func renderColumnUsageTable(md *markdown.MarkDown, summary TableSummary) {
	headers := []string{"Column", "Filter %", "Grouping %", "Join %"}
	var rows [][]string
	for colName, usage := range summary.GetColumns() {
		rows = append(rows, []string{
			colName,
			fmt.Sprintf("%.2f%%", usage.FilterPercentage),
			fmt.Sprintf("%.2f%%", usage.GroupingPercentage),
			fmt.Sprintf("%.2f%%", usage.JoinPercentage),
		})
	}
	md.PrintTable(headers, rows)
}

func (g queryGraph) AddJoinPredicate(key graphKey, pred operators.JoinPredicate) {
	g[key] = append(g[key], pred)
}

func renderJoinPredicatesUsage(md *markdown.MarkDown, summary *keys.Output) {
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
		md.PrintHeader("Table Relationships", 2)
	}
	for key, predicates := range g {
		md.Println("```")
		md.Printf("%s <> %s\n", key.Tbl1, key.Tbl2)
		md.Printf("|\n")
		for _, predicate := range predicates {
			md.Printf("|- %s\n", predicate.String())
		}
		md.Println("```")
		md.NewLine()
	}
}

// makeKey creates a graph key from two columns. The key is always sorted in ascending order.
func makeKey(lhs, rhs operators.Column) graphKey {
	if lhs.Table < rhs.Table {
		return graphKey{lhs.Table, rhs.Table}
	}

	return graphKey{rhs.Table, lhs.Table}
}

func createTableWriter(out io.Writer, cols []string) *tablewriter.Table {
	table := tablewriter.NewWriter(out)
	table.SetAutoFormatHeaders(false)
	table.SetHeader(cols)
	table.SetAutoWrapText(false)
	return table
}

func summarizeKeysQueries(queries *keys.Output) ([]TableSummary, []FailuresSummary) {
	tableSummaries := make(map[string]*TableSummary)
	tableUsageCounts := make(map[string]int)

	// First pass: collect all data and count occurrences
	for _, query := range queries.Queries {
		for _, table := range query.TableName {
			if _, exists := tableSummaries[table]; !exists {
				tableSummaries[table] = &TableSummary{
					Table:   table,
					Columns: make(map[string]ColumnUsage),
				}
			}
			tableUsageCounts[table] += query.UsageCount

			summarizeColumnUsage(table, tableSummaries, query)
			summarizeJoinPredicates(query.JoinPredicates, table, tableSummaries)
		}
	}

	// Second pass: calculate percentages
	for _, summary := range tableSummaries {
		count := tableUsageCounts[summary.Table]
		summary.QueryCount = count
		for colName, usage := range summary.Columns {
			countF := float64(count)
			usage.FilterPercentage = (usage.FilterPercentage / countF) * 100
			usage.GroupingPercentage = (usage.GroupingPercentage / countF) * 100
			usage.JoinPercentage = (usage.JoinPercentage / countF) * 100
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
	updateColumnUsage := func(columns any, usageType func(*ColumnUsage) *float64) {
		var colNames []string
		switch columns := columns.(type) {
		case []operators.Column:
			slice.Map(columns, func(col operators.Column) interface{} {
				colNames = append(colNames, col.String())
				return col
			})
		case []operators.ColumnUse:
			slice.Map(columns, func(col operators.ColumnUse) interface{} {
				colNames = append(colNames, col.Column.String())
				return col
			})
		}

		sort.Strings(colNames)
		colNames = slices.Compact(colNames)
		for _, col := range colNames {
			if strings.HasPrefix(col, table+".") {
				colName := strings.TrimPrefix(col, table+".")
				usage := tableSummaries[table].Columns[colName]
				*usageType(&usage) += float64(query.UsageCount)
				tableSummaries[table].Columns[colName] = usage
			}
		}
	}

	updateColumnUsage(query.FilterColumns, func(cu *ColumnUsage) *float64 { return &cu.FilterPercentage })
	updateColumnUsage(query.GroupingColumns, func(cu *ColumnUsage) *float64 { return &cu.GroupingPercentage })
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
