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

package benchstat

import (
	"fmt"
	"io"
	"iter"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/quick"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"

	"github.com/vitessio/vt/go/keys"
)

type (
	// TracedQuery represents the structure of each element in the JSON file
	TracedQuery struct {
		Trace      Trace  `json:"Trace"`
		Query      string `json:"Query"`
		LineNumber string `json:"LineNumber"`
	}

	// Trace represents the recursive structure of the Trace field
	Trace struct {
		OperatorType       string  `json:"OperatorType"`
		Variant            string  `json:"Variant"`
		NoOfCalls          int     `json:"NoOfCalls"`
		AvgNumberOfRows    float64 `json:"AvgNumberOfRows"`
		MedianNumberOfRows float64 `json:"MedianNumberOfRows"`
		ShardsQueried      int     `json:"ShardsQueried"`
		Inputs             []Trace `json:"Inputs,omitempty"`
	}

	QuerySummary struct {
		Q TracedQuery
		RouteCalls,
		RowsSent,
		RowsInMemory,
		ShardsQueried int
	}

	TraceFile struct {
		Name string

		// Only one of these fields will be populated
		TracedQueries   []TracedQuery
		AnalysedQueries []keys.QueryAnalysisResult
	}
)

func Run(args []string) {
	traces := make([]TraceFile, len(args))
	for i, arg := range args {
		traces[i] = readTraceFile(arg)
	}

	firstTrace := traces[0]
	if len(traces) == 1 {
		if firstTrace.AnalysedQueries == nil {
			printTraceSummary(os.Stdout, terminalWidth(), highlightQuery, firstTrace)
		} else {
			printKeysSummary(os.Stdout, firstTrace)
		}
	} else {
		compareTraces(os.Stdout, terminalWidth(), highlightQuery, firstTrace, traces[1])
	}
}

func visit(trace Trace, f func(Trace)) {
	f(trace)
	for _, input := range trace.Inputs {
		visit(input, f)
	}
}

func summarizeTraces(file TraceFile) map[string]QuerySummary {
	summary := make(map[string]QuerySummary)
	for _, traceElement := range file.TracedQueries {
		summary[traceElement.Query] = summarizeTrace(traceElement)
	}
	return summary
}

func (trace *Trace) TotalRows() int {
	return int(trace.AvgNumberOfRows * float64(trace.NoOfCalls))
}

func summarizeTrace(t TracedQuery) QuerySummary {
	summary := QuerySummary{
		Q: t,
	}

	visit(t.Trace, func(trace Trace) {
		summary.ShardsQueried += trace.ShardsQueried
		switch trace.OperatorType {
		case "Route":
			summary.RouteCalls += trace.NoOfCalls
			summary.RowsSent += trace.TotalRows()
		case "Sort":
			if trace.Variant == "Memory" {
				summary.RowsInMemory += int(trace.AvgNumberOfRows)
			}
		case "Join":
			if trace.Variant == "HashJoin" {
				// HashJoin has to keep the LHS in memory
				summary.RowsInMemory += trace.Inputs[0].TotalRows()
			}
		}
	})

	return summary
}

func exit(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

const queryPrefix = "Query: "

func limitQueryLength(query string, termWidth int) string {
	// Process the query string
	processedQuery := strings.ReplaceAll(query, "\n", " ") // Replace newlines with spaces
	processedQuery = strings.TrimSpace(processedQuery)     // Trim leading/trailing spaces

	// Calculate available space for query
	availableSpace := termWidth - len(queryPrefix) - 3 // 3 for ellipsis

	if len(processedQuery) > availableSpace {
		processedQuery = processedQuery[:availableSpace] + "..."
	}
	return processedQuery
}

func printTraceSummary(out io.Writer, termWidth int, highLighter Highlighter, file TraceFile) {
	summary := summarizeTraces(file)
	for i, query := range file.TracedQueries {
		if i > 0 {
			fmt.Fprintln(out)
		}
		querySummary := summary[query.Query]
		printQuery(out, termWidth, highLighter, query, false)
		table := tablewriter.NewWriter(out)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{
			"Route Calls",
			"Rows Sent",
			"Rows In Memory",
			"Shards Queried",
		})
		table.Append([]string{
			strconv.Itoa(querySummary.RouteCalls),
			strconv.Itoa(querySummary.RowsSent),
			strconv.Itoa(querySummary.RowsInMemory),
			strconv.Itoa(querySummary.ShardsQueried),
		})
		table.Render()
	}
}

type Highlighter func(out io.Writer, query string) error

func highlightQuery(out io.Writer, query string) error {
	return quick.Highlight(out, query, "sql", "terminal", "monokai")
}

func noHighlight(out io.Writer, query string) error {
	_, err := fmt.Fprint(out, query)
	return err
}

func printQuery(out io.Writer, terminalWidth int, highLighter Highlighter, q TracedQuery, significant bool) {
	fmt.Fprintf(out, "%s", queryPrefix)
	err := highLighter(out, limitQueryLength(q.Query, terminalWidth))
	if err != nil {
		return
	}
	improved := ""
	if significant {
		improved = " (significant)"
	}
	fmt.Fprintf(out, "\nLine # %s%s\n", q.LineNumber, improved)
}

const significantChangeThreshold = 10

func terminalWidth() int {
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // default to 80 if we can't get the terminal width
	}
	return termWidth
}

func compareTraces(out io.Writer, termWidth int, highLighter Highlighter, file1, file2 TraceFile) {
	summary1 := summarizeTraces(file1)
	summary2 := summarizeTraces(file2)

	var significantChanges, totalQueries int
	var s1RouteCalls, s1DataSent, s1MemoryRows, s1ShardsQueried int
	var s2RouteCalls, s2DataSent, s2MemoryRows, s2ShardsQueried int

	for _, q := range file1.TracedQueries {
		s1, ok1 := summary1[q.Query]
		s2, ok2 := summary2[q.Query]
		if !ok1 || !ok2 {
			continue
		}
		totalQueries++

		table := tablewriter.NewWriter(out)
		table.SetHeader([]string{"Metric", file1.Name, file2.Name, "Diff", "% Change"})
		table.SetAutoFormatHeaders(false)

		m1 := compareMetric(table, "Route Calls", s1.RouteCalls, s2.RouteCalls)
		m2 := compareMetric(table, "Rows Sent", s1.RowsSent, s2.RowsSent)
		m3 := compareMetric(table, "Rows In Memory", s1.RowsInMemory, s2.RowsInMemory)
		m4 := compareMetric(table, "Shards Queried", s1.ShardsQueried, s2.ShardsQueried)

		// we introduce variables to make sure we don't shortcut the evaluation
		significant := m1 || m2 || m3 || m4
		if significant {
			significantChanges++
		}

		s1RouteCalls += s1.RouteCalls
		s1DataSent += s1.RowsSent
		s1MemoryRows += s1.RowsInMemory
		s1ShardsQueried += s1.ShardsQueried
		s2RouteCalls += s2.RouteCalls
		s2DataSent += s2.RowsSent
		s2MemoryRows += s2.RowsInMemory
		s2ShardsQueried += s2.ShardsQueried

		printQuery(out, termWidth, highLighter, s1.Q, significant)
		table.Render()
		fmt.Fprintln(out)
	}

	totalRouteCallsChange := float64(s2RouteCalls-s1RouteCalls) / float64(s1RouteCalls) * 100
	totalDataSentChange := float64(s2DataSent-s1DataSent) / float64(s1DataSent) * 100
	totalMemoryRowsChange := float64(s2MemoryRows-s1MemoryRows) / float64(s1MemoryRows) * 100
	totalShardsQueriedChange := float64(s2ShardsQueried-s1ShardsQueried) / float64(s1ShardsQueried) * 100

	// Print summary
	fmt.Fprintln(out, "Summary:")
	fmt.Fprintf(out, "- %d out of %d queries showed significant change\n", significantChanges, totalQueries)
	fmt.Fprintf(out, "- Average change in Route Calls: %.2f%%\n", totalRouteCallsChange)
	fmt.Fprintf(out, "- Average change in Data Sent: %.2f%%\n", totalDataSentChange)
	fmt.Fprintf(out, "- Average change in Rows In Memory: %.2f%%\n", totalMemoryRowsChange)
	fmt.Fprintf(out, "- Average change in Shards Queried: %.2f%%\n", totalShardsQueriedChange)
}

// compareMetric compares two metrics and appends the result to the table, returning true if the change is significant
func compareMetric(table *tablewriter.Table, metricName string, val1, val2 int) bool {
	diff := val2 - val1
	percentChange := float64(diff) / float64(val1) * 100
	percentChangeStr := fmt.Sprintf("%.2f%%", percentChange)
	if math.IsInf(percentChange, 0) {
		percentChangeStr = "∞%"
		percentChange = 0 // To not skew the average calculation
	}

	table.Append([]string{
		metricName,
		strconv.Itoa(val1),
		strconv.Itoa(val2),
		strconv.Itoa(diff),
		percentChangeStr,
	})

	return percentChange < -significantChangeThreshold
}

// printKeysSummary goes over all the analysed queries, gathers information about column usage per table,
// and prints this summary information to the output.
func printKeysSummary(out io.Writer, file TraceFile) {
	_, _ = fmt.Fprintf(out, "Summary from trace file %s\n", file.Name)
	tableSummaries := summarizeQueries(file.AnalysedQueries)
	for _, summary := range tableSummaries {
		table := tablewriter.NewWriter(out)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Column", "Filter %", "Grouping %", "Join %"})
		for colName, usage := range summary.GetColumns() {
			table.Append([]string{
				colName,
				fmt.Sprintf("%.2f%%", usage.FilterPercentage),
				fmt.Sprintf("%.2f%%", usage.GroupingPercentage),
				fmt.Sprintf("%.2f%%", usage.JoinPercentage),
			})
		}
		fmt.Fprintf(out, "Table: %s used in %d queries\n", summary.Table, summary.QueryCount)
		table.Render()
		_, _ = fmt.Fprintln(out)
	}
}

type ColumnUsage struct {
	FilterPercentage   float64
	GroupingPercentage float64
	JoinPercentage     float64
}

type TableSummary struct {
	Table      string
	QueryCount int
	Columns    map[string]ColumnUsage
}

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

func summarizeQueries(queries []keys.QueryAnalysisResult) []TableSummary {
	tableSummaries := make(map[string]*TableSummary)
	tableUsageCounts := make(map[string]int)

	// First pass: collect all data and count occurrences
	for _, query := range queries {
		for _, table := range query.TableName {
			if _, exists := tableSummaries[table]; !exists {
				tableSummaries[table] = &TableSummary{
					Table:   table,
					Columns: make(map[string]ColumnUsage),
				}
			}
			tableUsageCounts[table] += query.UsageCount

			updateColumnUsage := func(columns []string, usageType func(*ColumnUsage) *float64) {
				for _, col := range columns {
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
			updateColumnUsage(query.JoinColumns, func(cu *ColumnUsage) *float64 { return &cu.JoinPercentage })
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

	return result
}
