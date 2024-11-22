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

package transactions

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"vitess.io/vitess/go/vt/sqlparser"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
)

func TestRun(t *testing.T) {
	sb := &strings.Builder{}
	s := &state{
		parser: sqlparser.NewTestParser(),
		si:     &keys.SchemaInfo{},
		txs:    newTxSignatureMap(),
	}
	s.run(sb, Config{
		FileName: "../testdata/small-slow-query-log",
		Loader:   data.SlowQueryLogLoader{},
	})

	out, err := os.ReadFile("../testdata/small-slow-query-transactions.json")
	require.NoError(t, err)

	assert.Equal(t, string(out), sb.String())
}
