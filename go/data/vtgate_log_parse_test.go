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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVtGateQueryLog(t *testing.T) {
	gotQueries, err := VtGateLogLoader{NeedsBindVars: true}.Load("./testdata/vtgate.query.log")
	require.NoError(t, err)

	require.Len(t, gotQueries, 25)

	expect, err := os.ReadFile("./testdata/vtgate.query.log.parsed.bv.txt")
	require.NoError(t, err)

	var got []string
	for _, query := range gotQueries {
		got = append(got, query.Query)
	}

	require.Equal(t, string(expect), strings.Join(got, "\n"))
}

func TestParseVtGateQueryLogNoBindVars(t *testing.T) {
	gotQueries, err := VtGateLogLoader{NeedsBindVars: false}.Load("./testdata/vtgate.query.log")
	require.NoError(t, err)
	require.Len(t, gotQueries, 25)

	expect, err := os.ReadFile("./testdata/vtgate.query.log.parsed.txt")
	require.NoError(t, err)

	var got []string
	for _, query := range gotQueries {
		got = append(got, query.Query)
	}

	require.Equal(t, string(expect), strings.Join(got, "\n"))
}
