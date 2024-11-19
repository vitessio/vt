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

	"github.com/stretchr/testify/require"
)

func TestLoadSlowQueryLogWithMetadata(t *testing.T) {
	loader := SlowQueryLogLoader{}.Load("../testdata/slow_query_log")
	queries, err := makeSlice(loader)
	require.NoError(t, err)

	expected := []Query{
		{FirstWord: "/bin/mysqld,", Query: "/bin/mysqld, Version: 8.0.26 (Source distribution). started with:\nTcp port: 3306  Unix socket: /tmp/mysql.sock\nTime                 Id Command    Argument\nuse testdb;", Line: 1, Type: 0, QueryTime: 0.000153, LockTime: 6.3e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 0},
		{FirstWord: "SET", Query: "SET timestamp=1690891201;", Line: 8},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2343274;", Line: 9},
		{FirstWord: "FLUSH", Query: "FLUSH SLOW LOGS;", Line: 19, QueryTime: 0.005047, LockTime: 0, RowsSent: 0, RowsExamined: 0, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2343272;", Line: 24, QueryTime: 0.000162, LockTime: 6.7e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select s1_0.id, s1_0.code, s1_0.token, s1_0.date from stores s1_0 where s1_0.id=11393;", Line: 29, QueryTime: 0.000583, LockTime: 0.000322, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2343265;", Line: 34, QueryTime: 0.000148, LockTime: 6.2e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2343188;", Line: 39, QueryTime: 0.000159, LockTime: 6.5e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2343180;", Line: 44, QueryTime: 0.000152, LockTime: 6.3e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2343011;", Line: 49, QueryTime: 0.000149, LockTime: 6.1e-05, RowsSent: 666, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2342469;", Line: 54, QueryTime: 0.000153, LockTime: 6.2e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2342465;", Line: 59, QueryTime: 0.000151, LockTime: 6.2e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2342439;", Line: 64, QueryTime: 0.000148, LockTime: 6.1e-05, RowsSent: 1, RowsExamined: 731, Timestamp: 1690891201},
		{FirstWord: "select", Query: "select m1_0.id, m1_0.name, m1_0.value, m1_0.date from items m1_0 where m1_0.id=2342389;", Line: 69, QueryTime: 0.000163, LockTime: 6.7e-05, RowsSent: 1, RowsExamined: 1, Timestamp: 1690891201},
	}

	require.Equal(t, expected, queries)
}
