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

package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test utils getFileType function
func TestGetFileType(t *testing.T) {
	type testCase struct {
		filename      string
		expectedType  FileType
		expectedError string
	}
	testCases := []testCase{{
		filename:     "../testdata/keys-output/keys-log.json",
		expectedType: KeysFile,
	}, {
		filename:     "../testdata/dbinfo-output/sakila-dbinfo.json",
		expectedType: DBInfoFile,
	}, {
		filename:      "../testdata/query-logs/mysql.query.log",
		expectedType:  UnknownFile,
		expectedError: "'/' looking for beginning of value",
	}}
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			ft, err := GetFileType(tc.filename)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
			}
			assert.Equal(t, tc.expectedType, ft)
		})
	}
}
