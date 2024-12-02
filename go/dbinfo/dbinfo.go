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
	"encoding/json"
	"io"
	"os"
	"sort"

	"vitess.io/vitess/go/mysql"
)

type Config struct {
	VTParams mysql.ConnParams
}

func Run(cfg Config) error {
	return run(os.Stdout, cfg)
}

func run(out io.Writer, cfg Config) error {
	si, err := Get(cfg)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(si, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	return err
}

type TableColumn struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	KeyType    string `json:"keyType,omitempty"`
	IsNullable bool   `json:"isNullable,omitempty"`
	Extra      string `json:"extra,omitempty"`
}

type PrimaryKey struct {
	Columns []string `json:"columns"`
}

type Index struct {
	Name      string
	Columns   []string `json:"columns"`
	NonUnique bool     `json:"nonUnique,omitempty"`
}

type ForeignKey struct {
	ColumnName           string `json:"columnName"`
	ConstraintName       string `json:"constraintName"`
	ReferencedTableName  string `json:"referencedTableName"`
	ReferencedColumnName string `json:"referencedColumnName"`
}

type TableInfo struct {
	Name        string         `json:"name"`
	Rows        int            `json:"rows"`
	Columns     []*TableColumn `json:"columns"`
	PrimaryKey  *PrimaryKey    `json:"primaryKey,omitempty"`
	Indexes     []*Index       `json:"indexes,omitempty"`
	ForeignKeys []*ForeignKey  `json:"foreignKeys,omitempty"`
}

type Info struct {
	FileType        string            `json:"fileType"`
	Tables          []*TableInfo      `json:"tables"`
	GlobalVariables map[string]string `json:"globalVariables"`
}

func getTableSizes(dbh *DBHelper, tableMap map[string]*TableInfo) error {
	ts, err := dbh.getTableSizes()
	if err != nil {
		return err
	}

	for tableName, tableRows := range ts {
		ti, ok := tableMap[tableName]
		if !ok {
			ti = &TableInfo{
				Name: tableName,
			}
			tableMap[tableName] = ti
		}
		ti.Rows = tableRows
	}
	return nil
}

func getPrimaryKeys(dbh *DBHelper, tableMap map[string]*TableInfo) error {
	pks, err := dbh.getPrimaryKeys()
	if err != nil {
		return err
	}

	for tableName, pk := range pks {
		ti, ok := tableMap[tableName]
		if !ok {
			ti = &TableInfo{}
			tableMap[tableName] = ti
		}
		ti.PrimaryKey = &PrimaryKey{
			Columns: pk.columns,
		}
	}
	return nil
}

func getIndexes(dbh *DBHelper, tableMap map[string]*TableInfo) error {
	idxs, err := dbh.getIndexes()
	if err != nil {
		return err
	}

	for tableName, tidx := range idxs {
		ti, ok := tableMap[tableName]
		if !ok {
			ti = &TableInfo{}
			tableMap[tableName] = ti
		}
		for _, idx := range tidx.indexes {
			ti.Indexes = append(ti.Indexes, idx)
		}
	}
	return nil
}

func getForeignKeys(dbh *DBHelper, tableMap map[string]*TableInfo) error {
	fks, err := dbh.getForeignKeys()
	if err != nil {
		return err
	}

	for fkName, fk := range fks {
		ti, ok := tableMap[fkName]
		if !ok {
			ti = &TableInfo{}
			tableMap[fkName] = ti
		}
		ti.ForeignKeys = fk
	}
	return nil
}

func Get(cfg Config) (*Info, error) {
	vtParams := &mysql.ConnParams{
		Host:   cfg.VTParams.Host,
		Port:   cfg.VTParams.Port,
		Uname:  cfg.VTParams.Uname,
		Pass:   cfg.VTParams.Pass,
		DbName: cfg.VTParams.DbName,
	}

	var tableInfo []*TableInfo
	tableMap := make(map[string]*TableInfo)

	dbh := NewDBHelper(vtParams)

	globalVariables, err := dbh.getGlobalVariables()
	if err != nil {
		return nil, err
	}

	if err := getTableSizes(dbh, tableMap); err != nil {
		return nil, err
	}

	if err := getPrimaryKeys(dbh, tableMap); err != nil {
		return nil, err
	}

	if err := getIndexes(dbh, tableMap); err != nil {
		return nil, err
	}

	if err := getForeignKeys(dbh, tableMap); err != nil {
		return nil, err
	}

	for tableName := range tableMap {
		tableInfo = append(tableInfo, tableMap[tableName])
	}
	sort.Slice(tableInfo, func(i, j int) bool {
		return tableInfo[i].Name < tableInfo[j].Name
	})

	dbInfo := &Info{
		FileType:        "dbinfo",
		Tables:          tableInfo,
		GlobalVariables: globalVariables,
	}
	return dbInfo, nil
}

func Load(fileName string) (*Info, error) {
	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	var si Info
	err = json.Unmarshal(b, &si)
	if err != nil {
		return nil, err
	}
	return &si, nil
}
