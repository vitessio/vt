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

package reference

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/schema"
)

type Config struct {
	KeysOutputFile string
	SchemaInfoFile string

	Loader data.Loader
}

func Run(cfg Config) error {
	return run(os.Stdout, cfg)
}

func Find(cfg Config) (*ReferenceInfo, error) {
	ri, err := GetReferenceInfo(cfg)
	if err != nil {
		return nil, err
	}
	totalJoins := 0
	for _, ts := range ri.TableSummaries {
		totalJoins += ts.JoinCount
	}

	thresholdJoins := totalJoins / 20 // 5%
	writePercentage := func(ts *TableSummary) float64 {
		return float64(ts.NumWrites) / float64(ts.NumWrites+ts.NumReads)
	}
	writePercentageThreshold := 1 / 100.0 // 1%
	tableCountThreshold := 1000
	for _, ts := range ri.TableSummaries {
		tableName := strings.Trim(ts.TableName, "'`\"")
		numRows := ri.TableRows[tableName]
		if ts.JoinCount > thresholdJoins && writePercentage(ts) < writePercentageThreshold && numRows < tableCountThreshold {
			ri.ChosenTables = append(ri.ChosenTables, tableName)
		} else {
			fmt.Printf("Table: %s, Reads: %d, Writes: %d, Joins: %d, Rows: %d\n",
				ts.TableName, ts.NumReads, ts.NumWrites, ts.JoinCount, ri.TableRows[tableName])
		}
	}
	return ri, nil
}

type TableInfo struct {
	Name      string
	NumWrites int
	NumReads  int
	JoinCount int
	Rows      int
}
type ReferenceOutput struct {
	Tables []TableInfo
}

func run(out io.Writer, cfg Config) error {
	ri, err := Find(cfg)
	if err != nil {
		return err
	}
	ro := ReferenceOutput{}
	for _, table := range ri.ChosenTables {
		ts := ri.TableSummaries[table]
		ro.Tables = append(ro.Tables, TableInfo{
			Name:      table,
			NumWrites: ts.NumWrites,
			NumReads:  ts.NumReads,
			JoinCount: ts.JoinCount,
			Rows:      ri.TableRows[table],
		})
	}
	b, err := json.MarshalIndent(ro, "", "  ")
	if err != nil {
		return err
	}
	out.Write(b)
	return nil
}

type TableSummary struct {
	TableName string
	NumReads  int
	NumWrites int
	JoinCount int
}

func (ts TableSummary) String() string {
	return fmt.Sprintf("Table: %s, Reads: %d, Writes: %d, Joins: %d", ts.TableName, ts.NumReads, ts.NumWrites, ts.JoinCount)
}

type ReferenceInfo struct {
	TableSummaries map[string]*TableSummary
	ChosenTables   []string
	TableRows      map[string]int
}

func NewReferenceInfo() *ReferenceInfo {
	return &ReferenceInfo{
		TableSummaries: make(map[string]*TableSummary),
		TableRows:      make(map[string]int),
	}
}

func GetReferenceInfo(cfg Config) (*ReferenceInfo, error) {
	ri := NewReferenceInfo()
	keysConfig := keys.Config{
		FileName: cfg.KeysOutputFile,
		Loader:   cfg.Loader,
	}
	keysOutput, err := keys.GetKeysInfo(keysConfig)
	if err != nil {
		return nil, err
	}
	getRit := func(table string) *TableSummary {
		table = strings.Trim(table, "'`\"")
		summary, ok := ri.TableSummaries[table]
		if !ok {
			summary = &TableSummary{
				TableName: table,
			}
			ri.TableSummaries[table] = summary
		}
		return summary
	}

	for _, query := range keysOutput.Queries {
		usageCount := query.UsageCount
		isWrite := false
		isRead := false
		st := strings.ToLower(query.StatementType)
		switch st {
		case "select":
			isRead = true
		case "insert", "update", "delete":
			isWrite = true
		default:
			continue
		}

		for _, table := range query.TableNames {

			rit := getRit(table)
			if isRead {
				rit.NumReads += usageCount
			}
			if isWrite {
				rit.NumWrites += usageCount
			}
		}

		for _, pred := range query.JoinPredicates {
			rit1 := getRit(pred.LHS.Table)
			rit2 := getRit(pred.RHS.Table)
			rit1.JoinCount += usageCount
			rit2.JoinCount += usageCount
		}
	}

	si, err := schema.Load(cfg.SchemaInfoFile)
	if err != nil {
		return nil, err
	}
	for _, table := range ri.TableSummaries {
		for _, table2 := range si.Tables {
			t := strings.Trim(table.TableName, "'`\"")
			t2 := strings.Trim(table2.Name, "'`\"")
			if t == t2 {
				ri.TableRows[t] = table2.Rows
			}
		}
	}
	return ri, nil
}
