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

package summarize

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test utils getFileType function
func TestGetFileType(t *testing.T) {
	type testCase struct {
		filename      string
		expectedType  fileType
		expectedError string
	}
	testCases := []testCase{
		{
			filename:     "../testdata/keys-log.json",
			expectedType: keysFile,
		},
		{
			filename:     "../testdata/sakila-dbinfo.json",
			expectedType: dbInfoFile,
		},
		{
			filename:      "../testdata/mysql.query.log",
			expectedType:  unknownFile,
			expectedError: "error reading token: invalid character '/' looking for beginning of value",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			ft, err := getFileType(tc.filename)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
			}
			assert.Equal(t, tc.expectedType, ft)
		})
	}
}
