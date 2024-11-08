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

	"github.com/stretchr/testify/require"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/typ"
)

func TestKeys(t *testing.T) {
	sb := &strings.Builder{}
	cfg := Config{
		FileName: "../../t/tpch_failing_queries.test",
		Loader:   data.SQLScriptLoader{},
	}
	err := run(sb, cfg)
	require.NoError(t, err)

	out, err := os.ReadFile("../summarize/testdata/keys-log.json")
	require.NoError(t, err)

	require.Equal(t, string(out), sb.String())
}

func TestKeysNonAuthoritativeTable(t *testing.T) {
	q := data.Query{
		Query: "select id from user where id = 20",
		Type:  typ.Query,
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
