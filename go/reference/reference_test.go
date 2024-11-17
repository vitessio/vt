package reference

import (
	"github.com/stretchr/testify/require"
	"github.com/vitessio/vt/go/data"
	"testing"
)

func TestReference(t *testing.T) {
	cfg := Config{
		FileName: "../../t/sakila/sakila.test",
		Loader:   data.SQLScriptLoader{},
	}

	ri, err := Find(cfg)
	require.NoError(t, err)
	require.NotNil(t, ri)
	require.NotEmpty(t, ri.TableSummaries)
	validReferenceTables := []string{"actor", "address", "category", "city", "country", "film", "language", "staff"}
	for _, table := range ri.ChosenTables {
		require.Containsf(t, validReferenceTables, table, "table %s is not a valid reference table", table)
	}
}
