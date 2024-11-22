package summarize

import (
	"testing"

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
			expectedType: unknownFile,
		},
		{
			filename:     "../testdata/sakila-schema-info.json",
			expectedType: dbInfoFile,
		},
		{
			filename:      "../testdata/mysql.query.log",
			expectedType:  unknownFile,
			expectedError: "Error reading token",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			ft, err := getFileType(tc.filename)
			if tc.expectedError != "" {
				require.Error(t, err)
			}
			if err != nil {
				require.Contains(t, err.Error(), tc.expectedError)
			}
			if ft != tc.expectedType {
				t.Errorf("Expected type: %v, got: %v", tc.expectedType, ft)
			}
		})
	}
}
