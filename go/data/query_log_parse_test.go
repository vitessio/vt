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

func TestParseMySQLQueryLog(t *testing.T) {
	loader := MySQLLogLoader{}.Load("../testdata/query-logs/mysql.query.log")
	gotQueries, err := makeSlice(loader)
	require.NoError(t, err)
	for _, query := range gotQueries {
		assert.NotZero(t, query.ConnectionID, query.Query)
	}
	require.Equal(t, 1517, len(gotQueries), "expected 1517 queries") //nolint:testifylint // too many elements for the output to be readable
}

func TestSmallSnippet(t *testing.T) {
	loader := MySQLLogLoader{}.Load("../testdata/query-logs/mysql.small-query.log")
	gotQueries, err := makeSlice(loader)
	require.NoError(t, err)
	expected := []Query{
		{
			Query:        "SET GLOBAL log_output = 'FILE'",
			Line:         4,
			Type:         SQLQuery,
			ConnectionID: 32,
		}, {
			Query:        "show databases",
			Line:         5,
			Type:         SQLQuery,
			ConnectionID: 32,
		}, {
			Query: `UPDATE _vt.schema_migrations
SET
	migration_status='queued',
	tablet='test_misc-0000004915',
	retries=retries + 1,
	tablet_failure=0,
	message='',
	stage='',
	cutover_attempts=0,
	ready_timestamp=NULL,
	started_timestamp=NULL,
	liveness_timestamp=NULL,
	cancelled_timestamp=NULL,
	completed_timestamp=NULL,
	last_cutover_attempt_timestamp=NULL,
	cleanup_timestamp=NULL
WHERE
	migration_status IN ('failed', 'cancelled')
	AND (
        tablet_failure=1
        AND migration_status='failed'
        AND retries=0
    )
LIMIT 1`,
			Line:         6,
			Type:         SQLQuery,
			ConnectionID: 24,
		},
	}

	require.Equal(t, expected, gotQueries)
}
