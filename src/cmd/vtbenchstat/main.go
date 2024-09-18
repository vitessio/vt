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
		printSummary(traces[0])
	} else {
		compareTraces(traces[0], traces[1])
	}
}

//func printSummary(trace TraceLog) {
//	fmt.Printf("Summary for %s:\n", trace.Filename)
//	table := tablewriter.NewWriter(os.Stdout)
//	table.SetHeader([]string{"Metric", "Value"})
//	table.Append([]string{"Total Queries", fmt.Sprintf("%d", trace.TotalQueries)})
//	table.Append([]string{"Average Time", fmt.Sprintf("%.2f ms", trace.AverageTime)})
//	// Add more metrics as needed
//	table.Render()
//}
//
//func compareTraces(trace1, trace2 TraceLog) {
//	fmt.Printf("Comparison between %s and %s:\n", trace1.Filename, trace2.Filename)
//	table := tablewriter.NewWriter(os.Stdout)
//	table.SetHeader([]string{"Metric", trace1.Filename, trace2.Filename, "Difference"})
//
//	metrics := []struct {
//		name string
//		v1   float64
//		v2   float64
//	}{
//		{"Total Queries", float64(trace1.TotalQueries), float64(trace2.TotalQueries)},
//		{"Average Time", trace1.AverageTime, trace2.AverageTime},
//		// Add more metrics as needed
//	}
//
//	for _, m := range metrics {
//		diff := m.v2 - m.v1
//		diffStr := fmt.Sprintf("%.2f", diff)
//		if diff > 0 {
//			diffStr = color.GreenString("+%s", diffStr)
//		} else if diff < 0 {
//			diffStr = color.RedString("%s", diffStr)
//		}
//		table.Append([]string{
//			m.name,
//			fmt.Sprintf("%.2f", m.v1),
//			fmt.Sprintf("%.2f", m.v2),
//			diffStr,
//		})
//	}
//
//	table.Render()
//}
