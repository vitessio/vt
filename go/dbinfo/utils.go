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

package dbinfo

import (
	"context"
	"fmt"

	"vitess.io/vitess/go/mysql"
)

type DBHelper struct {
	vtParams *mysql.ConnParams
}

func NewDBHelper(vtParams *mysql.ConnParams) *DBHelper {
	return &DBHelper{vtParams: vtParams}
}

func (dbh *DBHelper) GetConnection() (*mysql.Conn, func(), error) {
	vtConn, err := mysql.Connect(context.Background(), dbh.vtParams)
	if err != nil {
		return nil, nil, err
	}
	return vtConn, func() { vtConn.Close() }, nil
}

type tableSizes map[string]int

func (dbh *DBHelper) getTableSizes() (tableSizes, error) {
	vtConn, cancel, err := dbh.GetConnection()
	if err != nil {
		return nil, err
	}
	defer cancel()
	queryTableSizes := "SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema = '%s' and table_type = 'BASE TABLE'"
	qr, err := vtConn.ExecuteFetch(fmt.Sprintf(queryTableSizes, dbh.vtParams.DbName), -1, false)
	if err != nil {
		return nil, err
	}
	ts := make(tableSizes)
	for _, row := range qr.Rows {
		tableName := row[0].ToString()
		tableRows, _ := row[1].ToInt64()
		ts[tableName] = int(tableRows)
	}
	return ts, nil
}
