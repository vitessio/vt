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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTableSummary(t *testing.T) {
	expected := []string{
		"l_orderkey " + Join.String(),
		"l_orderkey " + Grouping.String(),
		"l_suppkey " + Join.String(),
		"l_suppkey " + JoinRange.String(),
		"l_commitdate " + WhereRange.String(),
		"l_receiptdate " + WhereRange.String(),
		"l_shipdate " + WhereRange.String(),
		"l_partkey " + Join.String(),
		"l_returnflag " + Where.String(),
		"l_shipmode " + WhereRange.String(),
		"l_shipmode " + Grouping.String(),
	}

	ts := TableSummary{
		Columns: map[ColumnInformation]ColumnUsage{
			{Name: "l_shipmode", Pos: WhereRange}:    {Percentage: 6},
			{Name: "l_receiptdate", Pos: WhereRange}: {Percentage: 28},
			{Name: "l_shipdate", Pos: WhereRange}:    {Percentage: 22},
			{Name: "l_orderkey", Pos: Grouping}:      {Percentage: 17},
			{Name: "l_orderkey", Pos: Join}:          {Percentage: 72},
			{Name: "l_suppkey", Pos: Join}:           {Percentage: 39},
			{Name: "l_shipmode", Pos: Grouping}:      {Percentage: 6},
			{Name: "l_returnflag", Pos: Where}:       {Percentage: 6},
			{Name: "l_partkey", Pos: Join}:           {Percentage: 17},
			{Name: "l_suppkey", Pos: JoinRange}:      {Percentage: 17},
			{Name: "l_commitdate", Pos: WhereRange}:  {Percentage: 28},
		},
	}

	var got []string
	//nolint:unused // Ignore unused variable check for this loop
	for ci, _ := range ts.GetColumns() {
		got = append(got, ci.String())
	}

	require.Equal(t, expected, got)
}

func TestSummarizeKeysFile(t *testing.T) {
	file := readTraceFile("testdata/keys-log.json")
	sb := &strings.Builder{}
	printKeysSummary(sb, file, time.Date(2024, time.January, 1, 1, 2, 3, 0, time.UTC))
	expected, err := os.ReadFile("testdata/keys-summary.md")
	require.NoError(t, err)
	require.Equal(t, string(expected), sb.String())
}
