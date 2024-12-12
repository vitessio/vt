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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizePlans(t *testing.T) {
	fnPlan, err := readPlanalyzeFile("../testdata/planalyze-output/bigger_slow_query_plan_report.json")
	require.NoError(t, err)

	fnKeys, err := readKeysFile("../testdata/keys-output/bigger_slow_query_log.json")
	require.NoError(t, err)

	sb := &strings.Builder{}
	now := time.Date(2024, time.January, 1, 1, 2, 3, 0, time.UTC)

	s, err := NewSummary("usage-count")
	require.NoError(t, err)

	err = fnPlan(s)
	require.NoError(t, err)

	err = fnKeys(s)
	require.NoError(t, err)

	err = compileSummary(s)
	require.NoError(t, err)

	err = s.PrintMarkdown(sb, now)
	require.NoError(t, err)

	expected, err := os.ReadFile("../testdata/summarize-output/bigger_slow_query_plan_report.md")
	require.NoError(t, err)
	assert.Equal(t, string(expected), sb.String())
	if t.Failed() {
		_ = os.Mkdir("../testdata/expected", 0o755)
		_ = os.WriteFile("../testdata/expected/bigger_slow_query_plan_report.md", []byte(sb.String()), 0o644)
	}
}
