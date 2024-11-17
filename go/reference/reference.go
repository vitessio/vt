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
	"fmt"
	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/keys"
	"io"
	"os"
	"strings"
)

type Config struct {
	FileName         string
	ConnectionString string

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
	for _, ts := range ri.TableSummaries {
		if ts.JoinCount > thresholdJoins && writePercentage(ts) < writePercentageThreshold {
			ri.ChosenTables = append(ri.ChosenTables, ts.TableName)
		}
	}
	return ri, nil
}

func run(out io.Writer, cfg Config) error {
	ri, err := Find(cfg)
	if err != nil {
		return err
	}
	for _, table := range ri.ChosenTables {
		fmt.Fprintf(out, "%s:: %+v\n", table, ri.TableSummaries[table])
	}
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
}

func GetReferenceInfo(cfg Config) (*ReferenceInfo, error) {
	ri := &ReferenceInfo{
		TableSummaries: make(map[string]*TableSummary),
	}
	keysConfig := keys.Config{
		FileName: cfg.FileName,
		Loader:   cfg.Loader,
	}
	keysOutput, err := keys.GetKeysInfo(keysConfig)
	if err != nil {
		return nil, err
	}
	getRit := func(table string) *TableSummary {
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
	return ri, nil
}
