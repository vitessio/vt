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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableSummary(t *testing.T) {
	expected := []string{
		"l_orderkey " + Join.String() + " 72%",
		"l_orderkey " + Grouping.String() + " 17%",
		"l_suppkey " + Join.String() + " 39%",
		"l_suppkey " + JoinRange.String() + " 17%",
		"l_commitdate " + WhereRange.String() + " 28%",
		"l_receiptdate " + WhereRange.String() + " 28%",
		"l_shipdate " + WhereRange.String() + " 22%",
		"l_partkey " + Join.String() + " 17%",
		"l_returnflag " + Where.String() + " 6%",
		"l_shipmode " + WhereRange.String() + " 6%",
		"l_shipmode " + Grouping.String() + " 6%",
	}

	ts := TableSummary{
		ColumnUses: map[ColumnInformation]ColumnUsage{
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
	for ci, cu := range ts.GetColumns() {
		got = append(got, fmt.Sprintf("%s %.0f%%", ci.String(), cu.Percentage))
	}

	require.Equal(t, expected, got)
}

func TestSummarizeKeysFile(t *testing.T) {
	sb := &strings.Builder{}
	now := time.Date(2024, time.January, 1, 1, 2, 3, 0, time.UTC)

	fnKeys := readKeysFile("../testdata/keys-log.json")
	fnSchemaInfo := readDBInfoFile("../testdata/keys-schema-info.json")

	s := NewSummary("")

	err := fnKeys(s)
	require.NoError(t, err)

	err = fnSchemaInfo(s)
	require.NoError(t, err)

	s.PrintMarkdown(sb, now)

	expected, err := os.ReadFile("../testdata/keys-summary.md")
	require.NoError(t, err)
	assert.Equal(t, string(expected), sb.String())
	if t.Failed() {
		_ = os.WriteFile("../testdata/expected/keys-summary.md", []byte(sb.String()), 0o644)
	}
}

func TestSummarizeKeysWithHotnessFile(t *testing.T) {
	tests := []string{
		"usage-count",
		"total-rows-examined",
		"avg-rows-examined",
		"avg-time",
		"total-time",
	}

	for _, metric := range tests {
		t.Run(metric, func(t *testing.T) {
			fn := readKeysFile("../testdata/bigger_slow_query_log.json")
			sb := &strings.Builder{}
			now := time.Date(2024, time.January, 1, 1, 2, 3, 0, time.UTC)

			s := NewSummary(metric)

			err := fn(s)
			require.NoError(t, err)

			s.PrintMarkdown(sb, now)

			expected, err := os.ReadFile(fmt.Sprintf("../testdata/bigger_slow_log_%s.md", metric))
			require.NoError(t, err)
			assert.Equal(t, string(expected), sb.String())
			if t.Failed() {
				_ = os.Mkdir("../testdata/expected", 0o755)
				_ = os.WriteFile(fmt.Sprintf("../testdata/expected/bigger_slow_log_%s.md", metric), []byte(sb.String()), 0o644)
			}
		})
	}
}
