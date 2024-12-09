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

package data

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCSVQueryLog(t *testing.T) {
	one := 1
	zero := 0
	three := 3
	cfg := CSVConfig{
		Header:            true,
		QueryField:        2,
		TimestampField:    &one,
		QueryTimeField:    &zero,
		ConnectionIDField: &three,
	}

	loader := CSVLogLoader{Config: cfg}.Load("../testdata/csv.query.log")
	gotQueries, err := makeSlice(loader)
	require.NoError(t, err)

	require.Len(t, gotQueries, 10)

	expect, err := os.ReadFile("../testdata/csv.query.parsed.txt")
	require.NoError(t, err)

	var got []string
	for _, query := range gotQueries {
		got = append(got, formatCsv(query))
	}

	require.Equal(t, string(expect), strings.Join(got, "\n"))
}

func formatCsv(query Query) string {
	return fmt.Sprintf("%d:%s:%f:%d", query.ConnectionID, query.Query, query.QueryTime, query.Timestamp)
}
