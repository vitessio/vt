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
	"regexp"
	"strconv"
	"strings"

	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/test/endtoend/utils"
	querypb "vitess.io/vitess/go/vt/proto/query"
)

// fieldTypeRegexp splits a type name such as "UINT64" into its base "UINT" and its size "64".
var fieldTypeRegexp = regexp.MustCompile(`^([a-zA-Z]*)(\d*)$`)

// execAllowingAnyFieldSize runs the query on Vitess and MySQL and compares the results. It accepts a Vitess column
// type that differs from the MySQL column type only in size. MySQL can return a smaller integer type than Vitess, for
// example when the left side of a join returns one row.
// See https://github.com/vitessio/vitess/issues/16508#issuecomment-2710692111.
func (nqr *ComparingQueryRunner) execAllowingAnyFieldSize(query string) {
	mysqlQr, vtQr := nqr.comparer.ExecNoCompare(query)
	matchFieldSizes(vtQr, mysqlQr)

	_ = utils.CompareVitessAndMySQLResults(nqr.reporter, query, nqr.comparer.VtConn, vtQr, mysqlQr, utils.CompareOptions{})
}

// matchFieldSizes sets the type of each Vitess field to the type of the MySQL field when the two types differ only
// in size.
func matchFieldSizes(vtQr, mysqlQr *sqltypes.Result) {
	if vtQr == nil || mysqlQr == nil {
		return
	}

	for i, vtField := range vtQr.Fields {
		if i >= len(mysqlQr.Fields) {
			return
		}

		myField := mysqlQr.Fields[i]
		if sameTypeIgnoringSize(vtField.Type, myField.Type) {
			vtField.Type = myField.Type
		}
	}
}

// sameTypeIgnoringSize reports whether the Vitess type and the MySQL type differ only in size. A signed Vitess
// integer matches an unsigned MySQL integer when the Vitess integer is larger. All character types match each other,
// and all binary types match each other.
func sameTypeIgnoringSize(vtType, myType querypb.Type) bool {
	if vtType == myType {
		return true
	}

	vtBase, vtSize, ok := splitFieldType(vtType)
	if !ok {
		return false
	}

	myBase, mySize, ok := splitFieldType(myType)
	if !ok {
		return false
	}

	switch {
	case strings.HasPrefix(myBase, "U") && !strings.HasPrefix(vtBase, "U"):
		return myBase[1:] == vtBase && vtSize > mySize
	case isCharType(vtBase) && isCharType(myBase):
		return true
	case isBinaryType(vtBase) && isBinaryType(myBase):
		return true
	default:
		return vtBase == myBase
	}
}

// splitFieldType splits a type such as INT64 into its base name and its size. The size is zero for a type without a
// size, such as VARCHAR.
func splitFieldType(t querypb.Type) (base string, size int, ok bool) {
	matches := fieldTypeRegexp.FindStringSubmatch(t.String())
	if len(matches) != 3 {
		return "", 0, false
	}

	if matches[2] == "" {
		return matches[1], 0, true
	}

	size, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, false
	}

	return matches[1], size, true
}

// isCharType reports whether the base type name is a character type.
func isCharType(base string) bool {
	return base == "CHAR" || base == "VARCHAR" || base == "TEXT" || base == "LONGTEXT"
}

// isBinaryType reports whether the base type name is a binary type.
func isBinaryType(base string) bool {
	return base == "BINARY" || base == "VARBINARY" || base == "BLOB" || base == "LONGBLOB"
}
