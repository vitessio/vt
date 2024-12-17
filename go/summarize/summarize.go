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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/fatih/color"
	"golang.org/x/term"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/web"
)

type (
	traceSummary struct {
		Name          string
		TracedQueries []TracedQuery
	}
)

type SummaryOutput struct {
	Summary
	DateOfAnalysis string
}

type summaryWorker = func(s *Summary) error

func Run(files []string, hotMetric string, showGraph bool, outputFormat string, launchWebServer bool) {
	var traces []traceSummary
	var workers []summaryWorker

	if launchWebServer && outputFormat != "html" {
		fmt.Println("cannot use --web flag without --format=html")
		os.Exit(1)
	}

	for _, file := range files {
		typ, err := data.GetFileType(file)
		exitIfError(err)
		var w summarizer
		var t traceSummary
		switch typ {
		case data.DBInfoFile:
			w, err = readDBInfoFile(file)
		case data.TransactionFile:
			w, err = readTransactionFile(file)
		case data.TraceFile:
			t, err = readTracedFile(file)
		case data.KeysFile:
			w, err = readKeysFile(file)
		case data.PlanalyzeFile:
			w, err = readPlanalyzeFile(file)
		default:
			err = errors.New("unknown file type")
		}
		exitIfError(err)

		if w != nil {
			workers = append(workers, w)
			continue
		}

		traces = append(traces, t)
	}

	traceCount := len(traces)
	if traceCount <= 0 {
		s, err := printSummary(hotMetric, workers, outputFormat, launchWebServer)
		exitIfError(err)
		if showGraph {
			err := renderQueryGraph(s)
			exitIfError(err)
		}
		return
	}

	err := checkTraceConditions(traces, workers, hotMetric)
	exitIfError(err)

	switch traceCount {
	case 1:
		printTraceSummary(os.Stdout, terminalWidth(), highlightQuery, traces[0])
	case 2:
		compareTraces(os.Stdout, terminalWidth(), highlightQuery, traces[0], traces[1])
	}
}

func exitIfError(err error) {
	if err == nil {
		return
	}
	_, _ = color.New(color.FgRed).Fprintln(os.Stderr, err.Error())

	os.Exit(1)
}

func printSummary(hotMetric string, workers []summaryWorker, outputFormat string, launchWebServer bool) (*Summary, error) {
	s, err := NewSummary(hotMetric)
	if err != nil {
		return nil, err
	}
	for _, worker := range workers {
		err := worker(s)
		if err != nil {
			return nil, err
		}
	}
	outputFormat = strings.ToLower(outputFormat)
	switch outputFormat {
	case "html":
		summarizeOutput := SummaryOutput{
			Summary:        *s,
			DateOfAnalysis: time.Now().Format(time.DateTime),
		}

		if launchWebServer {
			err = launchInBrowser(summarizeOutput)
			if err != nil {
				return s, err
			}
		} else {
			html, err := web.RenderFile("summarize.html", "layout_standalone.html", summarizeOutput)
			if err != nil {
				return nil, err
			}
			_, err = io.Copy(os.Stdout, html)
			if err != nil {
				return nil, err
			}
		}
	case "markdown":
		// Print the response
		err = s.PrintMarkdown(os.Stdout, time.Now())
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown output format")
	}
	return s, nil
}

func launchInBrowser(summarizeOutput SummaryOutput) error {
	html, err := web.RenderFile("summarize.html", "layout.html", summarizeOutput)
	if err != nil {
		return err
	}
	tmpFile, err := writeToTempFile(html.Bytes())
	if err != nil {
		return err
	}

	ch := make(chan error)
	go func() {
		web.Run(8080)
		ch <- nil
	}()
	port := int64(8080) // FIXME: take this from flags
	url := fmt.Sprintf("http://localhost:%d/summarize?file=", port) + tmpFile.Name()
	err = exec.Command("open", url).Start()
	if err != nil {
		fmt.Println("Error launching browser:", err)
		return err
	}
	fmt.Println("URL launched in default browser:", url)
	return <-ch
}

func writeToTempFile(summaryJSON []byte) (*os.File, error) {
	tmpFile, err := os.CreateTemp("/tmp/", "vt-summary-*.json")
	if err != nil {
		fmt.Println("Error creating temp file:", err)
		return nil, err
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(string(summaryJSON))
	if err != nil {
		fmt.Println("Error writing to temp file:", err)
		return nil, err
	}

	return tmpFile, err
}

func checkTraceConditions(traces []traceSummary, workers []summaryWorker, hotMetric string) error {
	if len(workers) > 0 {
		return errors.New("trace files cannot be mixed with other file types")
	}
	if len(traces) > 2 {
		return errors.New("can only summarize up to two trace files at once")
	}
	if hotMetric != "" {
		return errors.New("hotMetric flag is only supported for 'vt keys' output")
	}
	return nil
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
