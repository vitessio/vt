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
		"l_orderkey/" + Join.String() + " 72%",
		"l_orderkey/" + Grouping.String() + " 17%",
		"l_suppkey/" + Join.String() + " 39%",
		"l_suppkey/" + JoinRange.String() + " 17%",
		"l_commitdate/" + WhereRange.String() + " 28%",
		"l_receiptdate/" + WhereRange.String() + " 28%",
		"l_shipdate/" + WhereRange.String() + " 22%",
		"l_partkey/" + Join.String() + " 17%",
		"l_returnflag/" + Where.String() + " 6%",
		"l_shipmode/" + WhereRange.String() + " 6%",
		"l_shipmode/" + Grouping.String() + " 6%",
	}

	ts := TableSummary{
		ColumnUses: map[string]ColumnUsage{
			(&ColumnInformation{Name: "l_shipmode", Pos: WhereRange}).String():    {Percentage: 6},
			(&ColumnInformation{Name: "l_receiptdate", Pos: WhereRange}).String(): {Percentage: 28},
			(&ColumnInformation{Name: "l_shipdate", Pos: WhereRange}).String():    {Percentage: 22},
			(&ColumnInformation{Name: "l_orderkey", Pos: Grouping}).String():      {Percentage: 17},
			(&ColumnInformation{Name: "l_orderkey", Pos: Join}).String():          {Percentage: 72},
			(&ColumnInformation{Name: "l_suppkey", Pos: Join}).String():           {Percentage: 39},
			(&ColumnInformation{Name: "l_shipmode", Pos: Grouping}).String():      {Percentage: 6},
			(&ColumnInformation{Name: "l_returnflag", Pos: Where}).String():       {Percentage: 6},
			(&ColumnInformation{Name: "l_partkey", Pos: Join}).String():           {Percentage: 17},
			(&ColumnInformation{Name: "l_suppkey", Pos: JoinRange}).String():      {Percentage: 17},
			(&ColumnInformation{Name: "l_commitdate", Pos: WhereRange}).String():  {Percentage: 28},
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

	fnKeys, err := readKeysFile("../testdata/keys-output/keys-log.json")
	require.NoError(t, err)

	fnSchemaInfo, err := readDBInfoFile("../testdata/dbinfo-output/keys-schema-info.json")
	require.NoError(t, err)

	s, err := NewSummary("")
	require.NoError(t, err)

	err = fnKeys(s)
	require.NoError(t, err)

	err = fnSchemaInfo(s)
	require.NoError(t, err)

	err = s.PrintMarkdown(sb, now)
	require.NoError(t, err)

	expected, err := os.ReadFile("../testdata/summarize-output/keys-summary.md")
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
			fn, err := readKeysFile("../testdata/keys-output/bigger_slow_query_log.json")
			require.NoError(t, err)
			sb := &strings.Builder{}
			now := time.Date(2024, time.January, 1, 1, 2, 3, 0, time.UTC)

			s, err := NewSummary(metric)
			require.NoError(t, err)

			err = fn(s)
			require.NoError(t, err)

			err = compileSummary(s)
			require.NoError(t, err)

			err = s.PrintMarkdown(sb, now)
			require.NoError(t, err)

			expected, err := os.ReadFile(fmt.Sprintf("../testdata/summarize-output/bigger_slow_log_%s.md", metric))
			require.NoError(t, err)
			assert.Equal(t, string(expected), sb.String())
			if t.Failed() {
				_ = os.Mkdir("../testdata/expected", 0o755)
				_ = os.WriteFile(fmt.Sprintf("../testdata/expected/bigger_slow_log_%s.md", metric), []byte(sb.String()), 0o644)
			}
		})
	}
}
