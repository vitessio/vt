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

package benchstat

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func initFile(t *testing.T, firstToken rune) *os.File {
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(f.Name(), []byte(fmt.Sprintf("%c{\"value\": 1}", firstToken)), 0o600))
	return f
}

func TestGetDecoderAndDelim(t *testing.T) {
	tests := []struct {
		name      string
		wantDelim rune
	}{
		{
			name:      "[ delim",
			wantDelim: '[',
		},
		{
			name:      "{ delim",
			wantDelim: '{',
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := initFile(t, tt.wantDelim)
			defer f.Close()

			_, delim := getDecoderAndDelim(f)
			require.Equal(t, json.Delim(tt.wantDelim), delim)
		})
	}
}
