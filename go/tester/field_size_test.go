/*
Copyright 2026 The Vitess Authors.

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
	"testing"

	"github.com/stretchr/testify/assert"
	"vitess.io/vitess/go/sqltypes"
	querypb "vitess.io/vitess/go/vt/proto/query"
)

// TestSameTypeIgnoringSize checks which Vitess and MySQL type pairs differ only in size.
func TestSameTypeIgnoringSize(t *testing.T) {
	tests := []struct {
		name   string
		vtType querypb.Type
		myType querypb.Type
		want   bool
	}{
		{"equal types", sqltypes.Int64, sqltypes.Int64, true},
		{"larger vitess int", sqltypes.Int64, sqltypes.Int32, true},
		{"smaller vitess int", sqltypes.Int32, sqltypes.Int64, true},
		{"larger signed vitess int for unsigned mysql int", sqltypes.Int64, sqltypes.Uint32, true},
		{"equal size signed vitess int for unsigned mysql int", sqltypes.Int32, sqltypes.Uint32, false},
		{"unsigned vitess int for signed mysql int", sqltypes.Uint64, sqltypes.Int32, false},
		{"char types", sqltypes.VarChar, sqltypes.Text, true},
		{"binary types", sqltypes.VarBinary, sqltypes.Blob, true},
		{"char and binary types", sqltypes.VarChar, sqltypes.VarBinary, false},
		{"different types", sqltypes.Int64, sqltypes.Timestamp, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sameTypeIgnoringSize(tt.vtType, tt.myType))
		})
	}
}

// TestMatchFieldSizes checks that Vitess fields take the MySQL type only when the types differ only in size.
func TestMatchFieldSizes(t *testing.T) {
	vtQr := &sqltypes.Result{Fields: []*querypb.Field{
		{Name: "a", Type: sqltypes.Int64},
		{Name: "b", Type: sqltypes.Int64},
	}}
	mysqlQr := &sqltypes.Result{Fields: []*querypb.Field{
		{Name: "a", Type: sqltypes.Int32},
		{Name: "b", Type: sqltypes.Timestamp},
	}}

	matchFieldSizes(vtQr, mysqlQr)

	assert.Equal(t, sqltypes.Int32, vtQr.Fields[0].Type)
	assert.Equal(t, sqltypes.Int64, vtQr.Fields[1].Type)
}
