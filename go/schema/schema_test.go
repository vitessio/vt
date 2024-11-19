package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchema(t *testing.T) {
	si, err := Load("../testdata/sakila-schema-info.json")
	require.NoError(t, err)
	require.NotNil(t, si)
	require.NotEmpty(t, si.Tables)
	require.Equal(t, 16, len(si.Tables))
	var tables []string
	for _, table := range si.Tables {
		tables = append(tables, table.Name)
	}
	require.Contains(t, tables, "actor")
	require.NotContains(t, tables, "foo")
	for _, table := range si.Tables {
		require.NotEmpty(t, table.Name)
		switch table.Name {
		case "language":
			require.Equal(t, 6, table.Rows)
		case "film":
			require.Equal(t, 1000, table.Rows)
		default:
			require.Greater(t, table.Rows, 0, "table %s has no rows", table.Name)
		}
	}
}
