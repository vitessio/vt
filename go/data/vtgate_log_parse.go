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
)

type (
	bindVarsVtGate struct {
		Type  string `json:"type,omitempty"`
		Value any    `json:"value,omitempty"`
	}
	VtGateLogLoader struct {
		NeedsBindVars bool
	}

	vtgateLogReaderState struct {
		logReaderState
		NeedsBindVars bool
	}
)

func (vll VtGateLogLoader) Loadit(fileName string) IteratorLoader {
	reg := regexp.MustCompile(`\t"([^"]+)"\t(\{(?:[^{}]|(?:\{[^{}]*\}))*\}|"[^"]+")`)
	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return &errLoader{err: err}
	}

	scanner := bufio.NewScanner(fd)

	return &vtgateLogReaderState{
		logReaderState: logReaderState{
			scanner: scanner,
			reg:     reg,
			fd:      fd,
		},
		NeedsBindVars: vll.NeedsBindVars,
	}
}

func (s *vtgateLogReaderState) fail(err error) {
	s.err = err
}

func (s *vtgateLogReaderState) Next() (Query, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.err != nil {
		return Query{}, false
	}

	for s.scanner.Scan() {
		s.lineNumber++
		line := s.scanner.Text()

		if len(line) == 0 {
			continue
		}
		line = strings.ReplaceAll(line, "\\n", "")
		// Find the match
		match := s.reg.FindStringSubmatch(line)
		if len(match) <= 2 {
			s.fail(fmt.Errorf("line %d: cannot parse log: %s", s.lineNumber, line))
			return Query{}, false
		}

		query := match[1]
		if !s.NeedsBindVars {
			return Query{
				Query: query,
				Line:  s.lineNumber,
				Type:  QueryT,
			}, true
		}

		// If we care about bind variables (e.g. running 'trace') then we parse the query log
		// output into bindVarsVtGate, we then transform it into something the Vitess library
		// can understand (aka: map[string]*querypb.BindVariable), we then parse the query string
		// and add the bind variables to it.
		bindVarsRaw := match[2]
		bvs, err := getBindVariables(bindVarsRaw, s.lineNumber)
		if err != nil {
			s.fail(err)
			return Query{}, false
		}

		parsedQuery, err := addBindVarsToQuery(query, bvs)
		if err != nil {
			s.fail(err)
			return Query{}, false
		}

		return Query{
			Query: parsedQuery,
			Line:  s.lineNumber,
			Type:  QueryT,
		}, true
	}

	s.closed = true

	return Query{}, false
}

func (vll VtGateLogLoader) Load(fileName string) (queries []Query, err error) {
	loader := vll.Loadit(fileName)
	return makeSlice(loader)
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
	if strings.Contains(bindVarsRaw, "[REDACTED]") {
		return nil, fmt.Errorf("line %d: query has redacted bind variables, cannot parse them", lineNumber)
	}

	bv := map[string]bindVarsVtGate{}
	err := json.Unmarshal([]byte(bindVarsRaw), &bv)
	if err != nil {
		return nil, fmt.Errorf("line %d: error parsing bind variables: %v", lineNumber, err)
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
			return nil, fmt.Errorf("line %d: cannot parse tuple bind variables", lineNumber)
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
