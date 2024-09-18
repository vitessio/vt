package main

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
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
		AvgNumberOfRows    int     `json:"AvgNumberOfRows"`
		MedianNumberOfRows int     `json:"MedianNumberOfRows"`
		Inputs             []Trace `json:"Inputs,omitempty"`
	}

	Summary struct {
		RouteCalls int
		RowsSent   int
	}

	TraceFile []TracedQuery
)

//func main() {
//	if len(os.Args) != 3 {
//		fmt.Println("Usage: vtbenchstat <trace-file1> <trace-file2>")
//		os.Exit(1)
//	}
//	name1 := os.Args[1]
//	summary1 := summarizeTraces(readTraceFile(name1))
//	name2 := os.Args[2]
//	summary2 := summarizeTraces(readTraceFile(name2))
//	table := tablewriter.NewWriter(os.Stdout)
//	table.SetHeader([]string{"", name1, name2, "Diff", "% Change"})
//
//}

func visit(trace Trace, f func(Trace)) {
	f(trace)
	for _, input := range trace.Inputs {
		visit(input, f)
	}
}

func summarizeTraces(traces TraceFile) map[string]Summary {
	summary := make(map[string]Summary)
	for _, traceElement := range traces {
		summary[traceElement.Query] = summarizeTrace(traceElement.Trace)
	}
	return summary
}

func summarizeTrace(t Trace) Summary {
	var summary Summary

	visit(t, func(trace Trace) {
		summary.RouteCalls += trace.NoOfCalls
		summary.RowsSent += trace.AvgNumberOfRows * trace.NoOfCalls
	})

	return summary
}

func readTraceFile(fileName string) []TracedQuery {
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
	var elements []TracedQuery
	for decoder.More() {
		var element TracedQuery
		err := decoder.Decode(&element)
		if err != nil {
			panic(err.Error())
		}
		elements = append(elements, element)
	}

	// Read the closing bracket
	_, err = decoder.Token()
	if err != nil {
		panic(err.Error())
	}

	return elements
}

func printSummary(file TraceFile) {
	summary := summarizeTraces(file)

	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80 // default to 80 if we can't get the terminal width
	}
	for _, query := range file {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Route Calls", "Rows Sent"})
		table.Append([]string{strconv.Itoa(summary[query.Query].RouteCalls), strconv.Itoa(summary[query.Query].RowsSent)})
		// Process the query string
		processedQuery := strings.ReplaceAll(query.Query, "\n", " ") // Replace newlines with spaces
		processedQuery = strings.TrimSpace(processedQuery)           // Trim leading/trailing spaces

		// Calculate available space for query
		const queryPrefix = "Query: "
		availableSpace := termWidth - len(queryPrefix) - 3 // 3 for ellipsis

		if len(processedQuery) > availableSpace {
			processedQuery = processedQuery[:availableSpace] + "..."
		}

		fmt.Printf("%s%s\n", queryPrefix, processedQuery)
		table.Render()
		fmt.Println()
	}
}

func compareTraces(file, file2 TraceFile) {
	panic("not implemented")
}
