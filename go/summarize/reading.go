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

package summarize

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strconv"

	"github.com/vitessio/vt/go/keys"
	"github.com/vitessio/vt/go/schema"
)

func readTraceFile(fi fileInfo) traceSummary {
	switch fi.fileType {
	case traceFile:
		return readTracedQueryFile(fi.filename)
	default:
		panic("Unsupported file type")
	}
}

func readFile(fi fileInfo) (func(s *Summary) error, error) {
	switch fi.fileType {
	case keysFile:
		return readAnalysedQueryFile(fi.filename), nil
	case dbInfoFile:
		return readDBInfoFile(fi.filename), nil
	default:
		return nil, errors.New("unknown file format")
	}
}

func readTracedQueryFile(fileName string) traceSummary {
	c, err := os.ReadFile(fileName)
	if err != nil {
		exit("Error opening file: " + err.Error())
	}

	type traceOutput struct {
		FileType string        `json:"fileType"`
		Queries  []TracedQuery `json:"queries"`
	}
	var to traceOutput
	err = json.Unmarshal(c, &to)
	if err != nil {
		exit("Error parsing json: " + err.Error())
	}

	sort.Slice(to.Queries, func(i, j int) bool {
		a, err := strconv.Atoi(to.Queries[i].LineNumber)
		if err != nil {
			return false
		}
		b, err := strconv.Atoi(to.Queries[j].LineNumber)
		if err != nil {
			return false
		}
		return a < b
	})

	return traceSummary{
		Name:          fileName,
		TracedQueries: to.Queries,
	}
}

func readAnalysedQueryFile(fileName string) func(s *Summary) error {
	c, err := os.ReadFile(fileName)
	if err != nil {
		exit("Error opening file: " + err.Error())
	}

	var ko keys.Output
	err = json.Unmarshal(c, &ko)
	if err != nil {
		exit("Error parsing json: " + err.Error())
	}

	return func(s *Summary) error {
		s.analyzedFiles = append(s.analyzedFiles, fileName)
		summarizeKeysQueries(s, &ko)
		return nil
	}
}

func readDBInfoFile(fileName string) func(s *Summary) error {
	schemaInfo, err := schema.Load(fileName)
	if err != nil {
		panic(err)
	}

	return func(s *Summary) error {
		s.analyzedFiles = append(s.analyzedFiles, fileName)
		s.hasRowCount = true
		for _, ti := range schemaInfo.Tables {
			table := s.GetTable(ti.Name)
			if table == nil {
				table = &TableSummary{Table: ti.Name}
				s.AddTable(table)
			}
			table.RowCount = ti.Rows
		}
		return nil
	}
}
