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

package main

import (
	"encoding/json"
	"fmt"
	"github.com/alecthomas/chroma/quick"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
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
		Name    string
		Queries []TracedQuery
	}
)

func visit(trace Trace, f func(Trace)) {
	f(trace)
	for _, input := range trace.Inputs {
		visit(input, f)
	}
}

func summarizeTraces(file TraceFile) map[string]QuerySummary {
	summary := make(map[string]QuerySummary)
	for _, traceElement := range file.Queries {
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

func readTraceFile(fileName string) TraceFile {
	// Open the JSON file
	file, err := os.Open(fileName)
	if err != nil {
		exit("Error opening file: " + err.Error())
	}
	defer file.Close()

	// Create a decoder
	decoder := json.NewDecoder(file)

	// Read the opening bracket
	_, err = decoder.Token()
	if err != nil {
		exit("Error reading json: " + err.Error())
	}

	// Read the file contents
	var queries []TracedQuery
	for decoder.More() {
		var element TracedQuery
		err := decoder.Decode(&element)
		if err != nil {
			exit("Error reading json: " + err.Error())
		}
		queries = append(queries, element)
	}

	// Read the closing bracket
	_, err = decoder.Token()
	if err != nil {
		exit("Error reading json: " + err.Error())
	}

	sort.Slice(queries, func(i, j int) bool {
		a, err := strconv.Atoi(queries[i].LineNumber)
		if err != nil {
			return false
		}
		b, err := strconv.Atoi(queries[j].LineNumber)
		if err != nil {
			return false
		}
		return a < b
	})

	return TraceFile{
		Name:    fileName,
		Queries: queries,
	}
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

func printSummary(out io.Writer, termWidth int, highLighter Highlighter, file TraceFile) {
	summary := summarizeTraces(file)
	for i, query := range file.Queries {
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

	for _, q := range file1.Queries {
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
