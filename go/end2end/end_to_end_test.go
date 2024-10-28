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

/*
end2end contains end to end test for vt command line tool

it builds the vt binary and runs the tests and checks that outputs are as expected
*/
package end2end

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	projectRoot, err := filepath.Abs("../../")
	if err != nil {
		panic("failed to find project root: " + err.Error())
	}

	cmd := exec.Command("make", "build")
	cmd.Dir = projectRoot // Set working directory to project root

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Build output:\n%s\n", out.String())
		fmt.Printf("Build errors:\n%s\n", stderr.String())
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestApp(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"help"}, "Utils tools for testing, running and benchmarking Vitess"},
		{[]string{"keys", "t/demo.test"}, `"queryStructure"`},
		{[]string{
			"trace",
			"--backup-path", "go/end2end/testdata/backup",
			"--trace-file", "trace.log",
			"--vschema", "t/demo_vschema_shardkey.json",
			"t/demo.test",
		}, `ok! Ran 16 queries, 16 successfully`},
	}
	projectRoot, err := filepath.Abs("../../")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run("vt "+strings.Join(tt.args, " "), func(t *testing.T) {
			cmd := exec.Command("./vt", tt.args...)
			cmd.Dir = projectRoot
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			require.Contains(t, string(out), tt.want)
		})
	}
}
