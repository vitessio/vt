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
)

type (
	readingSummary struct {
		Name string

		// Only one of these fields will be populated
		TracedQueries   []TracedQuery // Set when analyzing a 'vt tester --trace' output
		AnalysedQueries *keys.Output  // Set when analyzing a 'vt keys' output
	}
)

func Run(args []string) {
	traces := make([]readingSummary, len(args))
	for i, arg := range args {
		traces[i] = readTraceFile(arg)
	}

	firstTrace := traces[0]
	if len(traces) == 1 {
		if firstTrace.AnalysedQueries == nil {
			printTraceSummary(os.Stdout, terminalWidth(), highlightQuery, firstTrace)
		} else {
			printKeysSummary(os.Stdout, firstTrace, time.Now())
		}
	} else {
		compareTraces(os.Stdout, terminalWidth(), highlightQuery, firstTrace, traces[1])
	}
}

func SummarizeKeysFile(fileName string) ([]TableSummary, error) {
	trace := readTraceFile(fileName)
	tableSummaries, failuresSummaries := summarizeKeysQueries(trace.AnalysedQueries)
	_ = failuresSummaries
	return tableSummaries, nil
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
