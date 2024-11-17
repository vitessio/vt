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

package keys

import (
	"os"
	"strings"
	"testing"
	"vitess.io/vitess/go/vt/sqlparser"

	"github.com/stretchr/testify/require"

	"github.com/vitessio/vt/go/data"
)

func TestKeys(t *testing.T) {
	cases := []struct {
		cfg          Config
		expectedFile string
	}{
		{
			cfg: Config{
				FileName: "../../t/tpch_failing_queries.test",
				Loader:   data.SQLScriptLoader{},
			},
			expectedFile: "../summarize/testdata/keys-log.json",
		},
		{
			cfg: Config{
				FileName: "../data/testdata/vtgate.query.log",
				Loader:   data.VtGateLogLoader{NeedsBindVars: false},
			},
			expectedFile: "../summarize/testdata/keys-log-vtgate.json",
		},
	}

	for _, tcase := range cases {
		t.Run(tcase.expectedFile, func(t *testing.T) {
			sb := &strings.Builder{}
			err := run(sb, tcase.cfg)
			require.NoError(t, err)

			out, err := os.ReadFile(tcase.expectedFile)
			require.NoError(t, err)

			require.Equal(t, string(out), sb.String())
		})
	}
}

func TestKeysNonAuthoritativeTable(t *testing.T) {
	q := data.Query{
		Query: "select id from user where id = 20",
		Type:  data.QueryT,
	}
	si := &schemaInfo{}
	ql := &queryList{
		queries: make(map[string]*QueryAnalysisResult),
		failed:  make(map[string]*QueryFailedResult),
	}
	process(q, si, ql)

	require.Len(t, ql.queries, 1)
	for _, result := range ql.queries {
		require.NotEmpty(t, result.FilterColumns)
	}
}

func TestGetUsageCount(t *testing.T) {
	type testCase struct {
		query      string
		usageCount int
	}

	cases := []testCase{
		{
			query:      "select /*vt+ VT_USAGE_COUNT=11 */ * from actor where first_name = 'Scarlett'",
			usageCount: 11,
		},
		{
			query:      "select /*vt+ VT_USAGE_COUNT=11.5 */ * from actor where first_name = 'Scarlett'",
			usageCount: 1,
		},
		{
			query:      "select /*vt+ VT_USAGE_COUNT=0 */ * from actor where first_name = 'Scarlett'",
			usageCount: 1,
		},
		{
			query:      "select  * from actor where first_name = 'Scarlett' and last_name = 'Johansson'",
			usageCount: 1,
		},
	}

	for _, tcase := range cases {
		ast, err := sqlparser.NewTestParser().Parse(tcase.query)
		require.NoError(t, err)
		usageCount := getUsageCount(ast)
		require.Equal(t, tcase.usageCount, usageCount)
	}
}
