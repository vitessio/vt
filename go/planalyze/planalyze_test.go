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

package planalyze

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	sb := &strings.Builder{}
	cfg := Config{
		VSchemaFile: "../testdata/planalyze-vschema.json",
	}

	err := run(sb, cfg, "../testdata/keys-output/bigger_slow_query_log.json")
	require.NoError(t, err)

	out, err := os.ReadFile("../testdata/planalyze-output/bigger_slow_query_plan_report.json")
	require.NoError(t, err)

	assert.Equal(t, string(out), sb.String())
	if t.Failed() {
		_ = os.WriteFile("../testdata/expected/bigger_slow_query_plan_report.json", []byte(sb.String()), 0o644)
	}
}
