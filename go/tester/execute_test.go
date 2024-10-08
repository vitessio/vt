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

package tester

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateShardRanges(t *testing.T) {
	tests := []struct {
		numberOfShards int
		expected       []string
	}{
		{1, []string{"-"}},
		{2, []string{"-80", "80-"}},
		{4, []string{"-40", "40-80", "80-c0", "c0-"}},
		{20, []string{"-0c", "0c-18", "18-24", "24-30", "30-3c", "3c-48", "48-54", "54-60", "60-6c", "6c-78", "78-84", "84-90", "90-9c", "9c-a8", "a8-b4", "b4-c0", "c0-cc", "cc-d8", "d8-e4", "e4-"}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d shards", tt.numberOfShards), func(t *testing.T) {
			result := generateShardRanges(tt.numberOfShards)
			assert.Equal(t, tt.expected, result)
		})
	}
}
