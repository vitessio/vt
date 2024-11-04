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
)

func TestKeys(t *testing.T) {
	sb := &strings.Builder{}
	err := run(sb, "../../t/tpch_failing_queries.test")
	require.NoError(t, err)

	out, err := os.ReadFile("../summarize/testdata/keys-log.json")
	require.NoError(t, err)

	require.Equal(t, string(out), sb.String())
}
