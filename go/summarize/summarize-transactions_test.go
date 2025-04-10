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

func TestSummarizeTransactionsFile(t *testing.T) {
	sb := &strings.Builder{}
	now := time.Date(2024, time.January, 1, 1, 2, 3, 0, time.UTC)

	fnTx, err := readTransactionFile("../testdata/transactions-output/small-slow-query-transactions.json")
	require.NoError(t, err)

	s, err := NewSummary("")
	require.NoError(t, err)

	err = fnTx(s)
	require.NoError(t, err)

	err = s.PrintMarkdown(sb, now)
	require.NoError(t, err)

	expected, err := os.ReadFile("../testdata/summarize-output/transactions-summary.md")
	require.NoError(t, err)
	assert.Equal(t, string(expected), sb.String())
	if t.Failed() {
		_ = os.WriteFile("../testdata/expected/transactions-summary.md", []byte(sb.String()), 0o644)
	}
}
