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

	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/markdown"
)

type (
	Summary struct {
		tables        []*TableSummary
		failures      []FailuresSummary
		hotQueries    []keys.QueryAnalysisResult
		hotQueryFn    getMetric
		analyzedFiles []string
		queryGraph    queryGraph
		hasRowCount   bool
	}

	traceSummary struct {
		Name          string
		TracedQueries []TracedQuery
	}
)

func NewSummary(hotMetric string) *Summary {
	return &Summary{
		queryGraph: make(queryGraph),
		hotQueryFn: getMetricForHotness(hotMetric),
	}
}

type summaryWorker = func(s *Summary) error

func Run(files []string, hotMetric string) {
	var traces []traceSummary
	var workers []summaryWorker

	for _, file := range files {
		typ, _ := getFileType(file)
		switch typ {
		case dbInfoFile:
			workers = append(workers, readDBInfoFile(file))
		case transactionFile:
			fmt.Printf("transaction file: %s\n", file)
		case traceFile:
			traces = append(traces, readTracedFile(file))
		case keysFile:
			workers = append(workers, readKeysFile(file))
		default:
			panic("Unknown file type")
		}
	}

	checkTraceConditions(traces, workers, hotMetric)

	if len(traces) == 2 {
		compareTraces(os.Stdout, terminalWidth(), highlightQuery, traces[0], traces[1])
		return
	} else if len(traces) == 1 {
		printTraceSummary(os.Stdout, terminalWidth(), highlightQuery, traces[0])
		return
	}

	s := NewSummary(hotMetric)
	for _, worker := range workers {
		err := worker(s)
		if err != nil {
			exit(err.Error())
		}
	}
	s.PrintMarkdown(os.Stdout, time.Now())
}

func (s *Summary) PrintMarkdown(out io.Writer, now time.Time) {
	md := &markdown.MarkDown{}
	msg := `# Query Analysis Report

**Date of Analysis**: %s  
**Analyzed Files**: ` + "%s" + `

`

	for i, file := range s.analyzedFiles {
		s.analyzedFiles[i] = "`" + file + "`"
	}
	md.Printf(msg, now.Format(time.DateTime), strings.Join(s.analyzedFiles, ", "))
	renderHotQueries(md, s.hotQueries, s.hotQueryFn)
	renderTableUsage(md, s.tables, s.hasRowCount)
	renderTablesJoined(md, s)
	renderFailures(md, s.failures)

	_, err := md.WriteTo(out)
	if err != nil {
		panic(err)
	}
}

func (s *Summary) GetTable(name string) *TableSummary {
	for _, table := range s.tables {
		if table.Table == name {
			return table
		}
	}
	return nil
}

func (s *Summary) AddTable(table *TableSummary) {
	s.tables = append(s.tables, table)
}

func checkTraceConditions(traces []traceSummary, workers []summaryWorker, hotMetric string) {
	if len(traces) == 0 {
		return
	}
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
