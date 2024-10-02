// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitessio/vitess-tester/go/typ"
	"testing"
)

func TestParseQueries(t *testing.T) {
	sql := "select * from t;"

	if q, err := ParseQueries(Query{Query: sql, Line: 1}); err == nil {
		assert.Equal(t, q[0].Type, typ.Query, fmt.Sprintf("Expected: %d, got: %d", typ.Query, q[0].Type))
		assert.Equal(t, q[0].Query, sql, fmt.Sprintf("Expected: %s, got: %s", sql, q[0].Query))
	} else {
		t.Fatalf("error is not nil. %v", err)
	}

	// invalid comment command style
	sql = "--abc select * from t;"
	_, err := ParseQueries(Query{Query: sql, Line: 1})
	assert.ErrorContains(t, err, "invalid command")
}
