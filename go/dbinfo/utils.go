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
	"strings"

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
	dbName := dbh.vtParams.DbName
	vtConn, cancel, err := dbh.GetConnection()
	if err != nil {
		return nil, err
	}
	defer cancel()
	queryTableSizes := "select table_name, table_rows from information_schema.tables where table_schema = '%s' and table_type = 'BASE TABLE'"
	query := fmt.Sprintf(queryTableSizes, dbName)
	qr, err := vtConn.ExecuteFetch(query, -1, false)
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

type tableColumns map[string][]*TableColumn

func (dbh *DBHelper) getColumnInfo() (tableColumns, error) {
	vtConn, cancel, err := dbh.GetConnection()
	if err != nil {
		return nil, err
	}
	defer cancel()
	queryColumnInfo := "select table_name, column_name, data_type, column_key, is_nullable, extra from information_schema.columns where table_schema = '%s'"
	query := fmt.Sprintf(queryColumnInfo, dbh.vtParams.DbName)
	qr, err := vtConn.ExecuteFetch(query, -1, false)
	if err != nil {
		return nil, err
	}
	tc := make(tableColumns)
	for _, row := range qr.Rows {
		tableName := row[0].ToString()
		columnName := row[1].ToString()
		dataType := strings.ToLower(row[2].ToString())
		columnKey := strings.ToLower(row[3].ToString())
		isNullable := row[4].ToString()
		extra := strings.ToLower(row[5].ToString())
		col := &TableColumn{
			Name:       columnName,
			Type:       dataType,
			KeyType:    columnKey,
			IsNullable: strings.EqualFold(isNullable, "YES"),
			Extra:      extra,
		}
		tc[tableName] = append(tc[tableName], col)
	}
	return tc, nil
}

func (dbh *DBHelper) getGlobalVariables() (map[string]string, error) {
	// Currently only use simple regex to match the variable names
	// If the variable name contains ".*" then it is treated as a regex, else exact match
	globalVariablesToFetch := []string{
		"binlog_format",
		"binlog_row_image",
		"log_bin",
		"gtid_mode",
	}

	vtConn, cancel, err := dbh.GetConnection()
	if err != nil {
		return nil, err
	}
	defer cancel()
	queryGlobalVars := "show global variables"
	qr, err := vtConn.ExecuteFetch(queryGlobalVars, -1, false)
	if err != nil {
		return nil, err
	}
	gv := make(map[string]string)
	for _, row := range qr.Rows {
		variable := row[0].ToString()
		value := row[1].ToString()
		for _, gvName := range globalVariablesToFetch {
			if variable == gvName {
				gv[variable] = value
			}
		}
	}
	return gv, nil
}
