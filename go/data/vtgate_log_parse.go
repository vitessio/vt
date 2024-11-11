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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"vitess.io/vitess/go/sqltypes"
	querypb "vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"

	"github.com/vitessio/vt/go/typ"
)

type bindVarsVtGate struct {
	Type  string `json:"type,omitempty"`
	Value any    `json:"value,omitempty"`
}

type VtGateLogLoader struct{}

func (VtGateLogLoader) Load(fileName string) (queries []Query, err error) {
	reg := regexp.MustCompile(`\t"([^"]+)"\t(\{(?:[^{}]|(?:\{[^{}]*\}))*\})`)

	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++

		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		line = strings.ReplaceAll(line, "\\n", "")

		// Find the match
		match := reg.FindStringSubmatch(line)
		if len(match) > 2 {
			query := match[1]
			bindVarsRaw := match[2]

			bvs, err := getBindVariables(bindVarsRaw, lineNumber)
			if err != nil {
				return nil, err
			}

			parsedQuery, err := addBindVarsToQuery(query, bvs)
			if err != nil {
				return nil, err
			}

			queries = append(queries, Query{
				Query: parsedQuery,
				Line:  lineNumber,
				Type:  typ.Query,
			})
		} else {
			return nil, fmt.Errorf("line %d: cannot parse log: %s", lineNumber, line)
		}
	}

	// Check for any errors that occurred during the scan
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return
}

func addBindVarsToQuery(query string, bvs map[string]*querypb.BindVariable) (string, error) {
	parser, err := sqlparser.New(sqlparser.Options{})
	if err != nil {
		return "", err
	}
	tree, err := parser.Parse(query)
	if err != nil {
		return "", err
	}

	buf := sqlparser.NewTrackedBuffer(nil)
	buf.Myprintf("%v", tree)
	pq := buf.ParsedQuery()
	return pq.GenerateQuery(bvs, nil)
}

func getBindVariables(bindVarsRaw string, lineNumber int) (map[string]*querypb.BindVariable, error) {
	bv := map[string]bindVarsVtGate{}
	err := json.Unmarshal([]byte(bindVarsRaw), &bv)
	if err != nil {
		return nil, fmt.Errorf("error parsing bind variables from line %d: %v", lineNumber, err)
	}

	bvProcessed := make(map[string]*querypb.BindVariable)
	for key, value := range bv {
		bvType := querypb.Type(querypb.Type_value[value.Type])

		var val []byte
		switch {
		case sqltypes.IsIntegral(bvType) || sqltypes.IsFloat(bvType):
			val = []byte(strconv.FormatFloat(value.Value.(float64), 'f', -1, 64))
		case bvType == sqltypes.Tuple:
			// the query log of vtgate does not list all the values for a tuple
			// instead it lists the following: "v2": {"type": "TUPLE", "value": "2 items"}
			panic("unsupported tuple in bindvars")
		default:
			val = []byte(value.Value.(string))
		}
		bvProcessed[key] = &querypb.BindVariable{
			Type:  bvType,
			Value: val,
		}
	}
	return bvProcessed, err
}
