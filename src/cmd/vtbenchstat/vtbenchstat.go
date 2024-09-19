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

func printSummary(file TraceFile) {
	summary := summarizeTraces(file)
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80 // default to 80 if we can't get the terminal width
	}
	for _, query := range file.Queries {
		querySummary := summary[query.Query]
		printQuery(query, termWidth, false)
		table := tablewriter.NewWriter(os.Stdout)
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
		fmt.Println()
	}
}

func printQuery(q TracedQuery, terminalWidth int, significant bool) {
	fmt.Printf("%s", queryPrefix)
	err := quick.Highlight(os.Stdout, limitQueryLength(q.Query, terminalWidth), "sql", "terminal", "monokai")
	if err != nil {
		return
	}
	improved := ""
	if significant {
		improved = " (significant)"
	}
	fmt.Printf("\nLine # %s%s\n", q.LineNumber, improved)
}

const significantChangeThreshold = 10

func compareTraces(file1, file2 TraceFile) {
	summary1 := summarizeTraces(file1)
	summary2 := summarizeTraces(file2)

	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80 // default to 80 if we can't get the terminal width
	}

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

		table := tablewriter.NewWriter(os.Stdout)
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

		printQuery(s1.Q, termWidth, significant)
		table.Render()
		fmt.Println()
	}

	totalRouteCallsChange := float64(s2RouteCalls-s1RouteCalls) / float64(s1RouteCalls) * 100
	totalDataSentChange := float64(s2DataSent-s1DataSent) / float64(s1DataSent) * 100
	totalMemoryRowsChange := float64(s2MemoryRows-s1MemoryRows) / float64(s1MemoryRows) * 100
	totalShardsQueriedChange := float64(s2ShardsQueried-s1ShardsQueried) / float64(s1ShardsQueried) * 100

	// Print summary
	fmt.Println("Summary:")
	fmt.Printf("- %d out of %d queries showed significant change\n", significantChanges, totalQueries)
	fmt.Printf("- Average change in Route Calls: %.2f%%\n", totalRouteCallsChange)
	fmt.Printf("- Average change in Data Sent: %.2f%%\n", totalDataSentChange)
	fmt.Printf("- Average change in Rows In Memory: %.2f%%\n", totalMemoryRowsChange)
	fmt.Printf("- Average change in Shards Queried: %.2f%%\n", totalShardsQueriedChange)
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
