package reference

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vitessio/vt/go/data"
)

func TestReference(t *testing.T) {
	cfg := Config{
		KeysOutputFile: "../../t/sakila/sakila.test",
		Loader:         data.SQLScriptLoader{},
		SchemaInfoFile: "../../t/sakila/sakila-schema-info.json",
	}

	ri, err := Find(cfg)
	require.NoError(t, err)
	require.NotNil(t, ri)
	require.NotEmpty(t, ri.TableSummaries)
	validReferenceTables := []string{"actor", "address", "category", "city", "country", "film", "language", "staff"}
	expectedTables := []string{"city", "language", "country", "address"}
	for _, table := range ri.ChosenTables {
		require.Containsf(t, validReferenceTables, table, "table %s is not a valid reference table", table)
	}
	sort.Strings(expectedTables)
	sort.Strings(ri.ChosenTables)
	require.EqualValuesf(t, expectedTables, ri.ChosenTables, "expected tables %v, got %v", expectedTables, ri.ChosenTables)
}
