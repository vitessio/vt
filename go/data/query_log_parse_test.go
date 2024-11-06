package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMySQLQueryLog(t *testing.T) {
	gotQueries, err := ParseMySQLQueryLog("./testdata/mysql.query.log")
	require.NoError(t, err)
	require.Len(t, gotQueries, 1516)
}
