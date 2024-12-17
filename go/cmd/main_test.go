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

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"help"}, "Utils tools for testing, running and benchmarking Vitess"},
		{[]string{"keys", "../../t/demo.test"}, `"queryStructure"`},
		{[]string{"keys", "--help"}, `Runs vexplain keys on all queries of the test file`},
		{[]string{"txs", "--help"}, `Analyze transactions on a query log`},
		{[]string{"test", "--help"}, `Test the given workload against both Vitess and MySQL`},
		{[]string{"trace", "--help"}, `Runs the given workload and does a`},
		{[]string{"summarize", "--help"}, `Compares and analyses a trace output`},
		{[]string{"dbinfo", "--help"}, `Loads info from the database including row counts`},
		{[]string{"planalyze", "--help"}, `Analyze the query plan`},
	}
	for _, tt := range tests {
		t.Run("vt  "+strings.Join(tt.args, " "), func(t *testing.T) {
			cmd := getRootCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			out := buf.String()
			require.NoError(t, err, out)
			require.Contains(t, out, tt.want)
		})
	}
}
