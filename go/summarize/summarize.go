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
	"os"
	"strings"
	"time"

	"github.com/alecthomas/chroma/quick"
	"golang.org/x/term"
)

type (
	traceSummary struct {
		Name          string
		TracedQueries []TracedQuery
	}
)

type summaryWorker = func(s *Summary) error

func Run(files []string, hotMetric string, showGraph bool) {
	var traces []traceSummary
	var workers []summaryWorker

	for _, file := range files {
		typ, err := getFileType(file)
		if err != nil {
			panic(err.Error())
		}
		switch typ {
		case dbInfoFile:
			workers = append(workers, readDBInfoFile(file))
		case transactionFile:
			workers = append(workers, readTransactionFile(file))
		case traceFile:
			traces = append(traces, readTracedFile(file))
		case keysFile:
			workers = append(workers, readKeysFile(file))
		default:
			panic("Unknown file type")
		}
	}

	traceCount := len(traces)
	if traceCount <= 0 {
		s := printSummary(hotMetric, workers)
		if showGraph {
			renderQueryGraph(s)
		}
		return
	}

	checkTraceConditions(traces, workers, hotMetric)
	switch traceCount {
	case 1:
		printTraceSummary(os.Stdout, terminalWidth(), highlightQuery, traces[0])
	case 2:
		compareTraces(os.Stdout, terminalWidth(), highlightQuery, traces[0], traces[1])
	}
}

func printSummary(hotMetric string, workers []summaryWorker) *Summary {
	s := NewSummary(hotMetric)
	for _, worker := range workers {
		err := worker(s)
		if err != nil {
			exit(err.Error())
		}
	}
	s.PrintMarkdown(os.Stdout, time.Now())
	return s
}

func checkTraceConditions(traces []traceSummary, workers []summaryWorker, hotMetric string) {
	if len(workers) > 0 {
		panic("Trace files cannot be mixed with other file types")
	}
	if len(traces) > 2 {
		panic("Can only summarize up to two trace files at once")
	}
	if hotMetric != "" {
		exit("hotMetric flag is only supported for 'vt keys' output")
	}
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

func terminalWidth() int {
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // default to 80 if we can't get the terminal width
	}
	return termWidth
}
