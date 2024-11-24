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

type TableInfo struct {
	Name string `json:"name"`
	Rows int    `json:"rows"`
}

type Info struct {
	FileType string      `json:"fileType"`
	Tables   []TableInfo `json:"tables"`
}

func Get(cfg Config) (*Info, error) {
	vtParams := &mysql.ConnParams{
		Host:   cfg.VTParams.Host,
		Port:   cfg.VTParams.Port,
		Uname:  cfg.VTParams.Uname,
		Pass:   cfg.VTParams.Pass,
		DbName: cfg.VTParams.DbName,
	}

	dbh := NewDBHelper(vtParams)
	ts, err := dbh.getTableSizes()
	if err != nil {
		return nil, err
	}

	var tableInfo []TableInfo
	tableMap := make(map[string]*TableInfo)

	for tableName, tableRows := range ts {
		tableMap[tableName] = &TableInfo{
			Name: tableName,
			Rows: tableRows,
		}
	}

	for tableName, _ := range tableMap {
		tableInfo = append(tableInfo, *tableMap[tableName])
	}
	dbInfo := &Info{
		Tables: tableInfo,
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
