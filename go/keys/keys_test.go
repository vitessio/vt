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

	"github.com/stretchr/testify/assert"
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
				Loader:   data.SlowQueryLogLoader{},
			},
			expectedFile: "../testdata/keys-log.json",
		},
		{
			cfg: Config{
				FileName: "../testdata/vtgate.query.log",
				Loader:   data.VtGateLogLoader{NeedsBindVars: false},
			},
			expectedFile: "../testdata/keys-log-vtgate.json",
		},
		{
			cfg: Config{
				FileName: "../testdata/slow_query_log",
				Loader:   data.SlowQueryLogLoader{},
			},
			expectedFile: "../testdata/slow-query-log.json",
		},
		{
			cfg: Config{
				FileName: "../testdata/bigger_slow_query_log.log",
				Loader:   data.SlowQueryLogLoader{},
			},
			expectedFile: "../testdata/bigger_slow_query_log.json",
		},
	}

	for _, tcase := range cases {
		t.Run(tcase.expectedFile, func(t *testing.T) {
			sb := &strings.Builder{}
			err := run(sb, tcase.cfg)
			require.NoError(t, err)

			out, err := os.ReadFile(tcase.expectedFile)
			require.NoError(t, err)

			assert.Equal(t, string(out), sb.String())
			if t.Failed() {
				_ = os.WriteFile(tcase.expectedFile+".correct", []byte(sb.String()), 0o644)
			}
		})
	}
}

func TestKeysNonAuthoritativeTable(t *testing.T) {
	q := data.Query{
		Query: "select id from user where id = 20",
		Type:  data.SQLQuery,
	}
	si := &SchemaInfo{}
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
