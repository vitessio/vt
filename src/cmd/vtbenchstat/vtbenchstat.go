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
	"strconv"
	"strings"
)

type (
	// TracedQuery represents the structure of each element in the JSON file
	TracedQuery struct {
		Trace Trace  `json:"Trace"`
		Query string `json:"Query"`
	}

	// Trace represents the recursive structure of the Trace field
	Trace struct {
		OperatorType       string  `json:"OperatorType"`
		Variant            string  `json:"Variant"`
		NoOfCalls          int     `json:"NoOfCalls"`
		AvgNumberOfRows    float64 `json:"AvgNumberOfRows"`
		MedianNumberOfRows float64 `json:"MedianNumberOfRows"`
		Inputs             []Trace `json:"Inputs,omitempty"`
	}

	Summary struct {
		RouteCalls int
		RowsSent   int
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

func summarizeTraces(file TraceFile) map[string]Summary {
	summary := make(map[string]Summary)
	for _, traceElement := range file.Queries {
		summary[traceElement.Query] = summarizeTrace(traceElement.Trace)
	}
	return summary
}

func summarizeTrace(t Trace) Summary {
	var summary Summary

	visit(t, func(trace Trace) {
		summary.RouteCalls += trace.NoOfCalls
		summary.RowsSent += int(trace.AvgNumberOfRows * float64(trace.NoOfCalls))
	})

	return summary
}

func readTraceFile(fileName string) TraceFile {
	// Open the JSON file
	file, err := os.Open(fileName)
	if err != nil {
		panic(err.Error())
	}
	defer file.Close()

	// Create a decoder
	decoder := json.NewDecoder(file)

	// Read the opening bracket
	_, err = decoder.Token()
	if err != nil {
		panic(err.Error())
	}

	// Read the file contents
	var queries []TracedQuery
	for decoder.More() {
		var element TracedQuery
		err := decoder.Decode(&element)
		if err != nil {
			panic(err.Error())
		}
		queries = append(queries, element)
	}

	// Read the closing bracket
	_, err = decoder.Token()
	if err != nil {
		panic(err.Error())
	}

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
	var summary map[string]Summary = summarizeTraces(file)
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80 // default to 80 if we can't get the terminal width
	}
	for _, query := range file.Queries {
		querySummary := summary[query.Query]
		printQuery(query.Query, termWidth)
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoFormatHeaders(false)
		table.SetHeader([]string{"Route Calls", "Rows Sent"})
		table.Append([]string{strconv.Itoa(querySummary.RouteCalls), strconv.Itoa(querySummary.RowsSent)})
		table.Render()
		fmt.Println()
	}
}

func printQuery(query string, terminalWidth int) {
	fmt.Printf("%s", queryPrefix)
	err := quick.Highlight(os.Stdout, limitQueryLength(query, terminalWidth), "sql", "terminal", "monokai")
	if err != nil {
		return
	}
	fmt.Println()
}

const significantChangeThreshold = 10

func compareTraces(file1, file2 TraceFile) {
	summary1 := summarizeTraces(file1)
	summary2 := summarizeTraces(file2)

	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80 // default to 80 if we can't get the terminal width
	}

	allQueries := make(map[string]struct{})
	for query := range summary1 {
		allQueries[query] = struct{}{}
	}
	for query := range summary2 {
		allQueries[query] = struct{}{}
	}

	var significantChanges, totalQueries int
	var totalRouteCallsChange, totalDataSentChange float64

	for query := range allQueries {
		s1, ok1 := summary1[query]
		s2, ok2 := summary2[query]
		if !(ok1 && ok2) {
			continue
		}
		totalQueries++

		printQuery(query, termWidth)
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Metric", file1.Name, file2.Name, "Diff", "% Change"})
		table.SetAutoFormatHeaders(false)

		routeCallsChange := compareMetric(table, "Route Calls", s1.RouteCalls, s2.RouteCalls)
		dataSentChange := compareMetric(table, "Rows Sent", s1.RowsSent, s2.RowsSent)

		totalRouteCallsChange += routeCallsChange
		if !math.IsNaN(dataSentChange) {
			totalDataSentChange += dataSentChange
		}

		if math.Abs(routeCallsChange) > significantChangeThreshold || math.Abs(dataSentChange) > significantChangeThreshold {
			significantChanges++
		}

		table.Render()
		fmt.Println()
	}

	// Print summary
	fmt.Println("Summary:")
	fmt.Printf("- %d out of %d queries showed significant change\n", significantChanges, totalQueries)
	fmt.Printf("- Average change in Route Calls: %.2f%%\n", totalRouteCallsChange/float64(totalQueries))
	fmt.Printf("- Average change in Data Sent: %.2f%%\n", totalDataSentChange/float64(totalQueries))
}

func compareMetric(table *tablewriter.Table, metricName string, val1, val2 int) float64 {
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

	return percentChange
}

func addMissingMetrics(table *tablewriter.Table, routeCalls, rowsSent int) {
	table.Append([]string{"Route Calls", strconv.Itoa(routeCalls), "N/A", "N/A", "N/A"})
	table.Append([]string{"Rows Sent", strconv.Itoa(rowsSent), "N/A", "N/A", "N/A"})
}
