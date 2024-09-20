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
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || len(args) > 2 {
		fmt.Println("Usage: vtbenchstat <trace_file1> [trace_file2]")
		os.Exit(1)
	}

	traces := make([]TraceFile, len(args))
	for i, arg := range args {
		traces[i] = readTraceFile(arg)
	}

	if len(traces) == 1 {
		printSummary(os.Stdout, terminalWidth(), highlightQuery, traces[0])
	} else {
		compareTraces(os.Stdout, terminalWidth(), highlightQuery, traces[0], traces[1])
	}
}
