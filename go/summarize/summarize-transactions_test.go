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

	fnTx := readTransactionFile("../testdata/small-slow-query-transactions.json")

	s := NewSummary("")

	err := fnTx(s)
	require.NoError(t, err)

	s.PrintMarkdown(sb, now)

	expected, err := os.ReadFile("../testdata/transactions-summary.md")
	require.NoError(t, err)
	assert.Equal(t, string(expected), sb.String())
	if t.Failed() {
		_ = os.WriteFile("../testdata/expected/transactions-summary.md", []byte(sb.String()), 0o644)
	}
}
