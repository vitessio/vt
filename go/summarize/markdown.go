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
	"slices"
	"sort"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"

	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/markdown"
)

func renderHotQueries(md *markdown.MarkDown, queries []keys.QueryAnalysisResult, metricReader getMetric) {
	if len(queries) == 0 {
		return
	}

	hasTime := false
	// Sort the queries in descending order of hotness
	sort.Slice(queries, func(i, j int) bool {
		if queries[i].QueryTime != 0 {
			hasTime = true
		}
		return metricReader(queries[i]) > metricReader(queries[j])
	})

	if !hasTime {
		return
	}

	md.PrintHeader("Top Queries", 2)

	// Prepare table headers and rows
	headers := []string{"Query ID", "Usage Count", "Total Query Time (ms)", "Avg Query Time (ms)", "Total Rows Examined"}
	var rows [][]string

	for i, query := range queries {
		queryID := fmt.Sprintf("Q%d", i+1)
		avgQueryTime := query.QueryTime / float64(query.UsageCount)
		rows = append(rows, []string{
			queryID,
			humanize.Comma(int64(query.UsageCount)),
			fmt.Sprintf("%.2f", query.QueryTime),
			fmt.Sprintf("%.2f", avgQueryTime),
			humanize.Comma(int64(query.RowsExamined)),
		})
	}

	// Print the table
	md.PrintTable(headers, rows)

	// After the table, list the full queries with their IDs
	md.PrintHeader("Query Details", 3)
	for i, query := range queries {
		queryID := fmt.Sprintf("Q%d", i+1)
		md.PrintHeader(queryID, 4)
		md.Println("```sql")
		md.Println(query.QueryStructure)
		md.Println("```")
		md.NewLine()
	}
}

func renderTableUsage(md *markdown.MarkDown, tableSummaries []*TableSummary, includeRowCount bool) {
	if len(tableSummaries) == 0 {
		return
	}

	sort.Slice(tableSummaries, func(i, j int) bool {
		if tableSummaries[i].UseCount() == tableSummaries[j].UseCount() {
			return tableSummaries[i].Table < tableSummaries[j].Table
		}
		return tableSummaries[i].UseCount() > tableSummaries[j].UseCount()
	})

	md.PrintHeader("Tables", 2)
	renderTableOverview(md, tableSummaries, includeRowCount)

	md.PrintHeader("Column Usage", 3)
	for _, summary := range tableSummaries {
		renderColumnUsageTable(md, summary)
	}
}

func renderTableOverview(md *markdown.MarkDown, tableSummaries []*TableSummary, includeRowCount bool) {
	headers := []string{"Table Name", "Reads", "Writes"}
	if includeRowCount {
		headers = append(headers, "Number of Rows")
	}
	var rows [][]string
	for _, summary := range tableSummaries {
		thisRow := []string{
			summary.Table,
			humanize.Comma(int64(summary.ReadQueryCount)),
			humanize.Comma(int64(summary.WriteQueryCount)),
		}
		if includeRowCount {
			thisRow = append(thisRow, humanize.Comma(int64(summary.RowCount)))
		}

		rows = append(rows, thisRow)
	}
	md.PrintTable(headers, rows)
}

func renderColumnUsageTable(md *markdown.MarkDown, summary *TableSummary) {
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

func renderTablesJoined(md *markdown.MarkDown, summary *Summary) {
	if len(summary.joins) == 0 {
		return
	}

	if len(summary.queryGraph) > 0 {
		md.PrintHeader("Tables Joined", 2)
	}

	md.Println("```")
	for _, join := range summary.joins {
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

func renderTransactions(md *markdown.MarkDown, transactions []TransactionSummary) {
	if len(transactions) == 0 {
		return
	}

	md.PrintHeader("Transaction Patterns", 2)

	for i, tx := range transactions {
		var tables []string
		for _, query := range tx.Queries {
			tables = append(tables, query.Table)
		}
		tables = uniquefy(tables)
		md.NewLine()
		md.PrintHeader(fmt.Sprintf("Pattern %d (Observed %d times)\n\n", i+1, tx.Count), 3)
		md.Printf("Tables Involved: %s\n", strings.Join(tables, ", "))
		md.PrintHeader("Query Patterns", 3)
		for i, query := range tx.Queries {
			md.Printf("%d. **%s** on `%s`  \n", i+1, strings.ToTitle(query.Type), query.Table)
			md.Printf("   Predicates: %s\n\n", strings.Join(query.Predicates, " AND "))
		}

		md.PrintHeader("Shared Predicate Values", 3)
		for idx, join := range tx.Joins {
			md.Printf("* Value %d applied to:\n", idx)
			for _, s := range join {
				md.Printf("  - %s\n", s)
			}
		}
		if i != len(transactions)-1 {
			md.Printf("---\n")
		}
	}
}

func uniquefy(s []string) []string {
	sort.Strings(s)
	return slices.Compact(s)
}
