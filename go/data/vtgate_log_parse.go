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
	"hash/fnv"
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
		uuidReg       *regexp.Regexp
	}
)

func (vll VtGateLogLoader) Load(fileName string) IteratorLoader {
	reg := regexp.MustCompile(`\t"([^"]+)"\t(\{(?:[^{}]|\{[^{}]*})*}|"[^"]+")`)
	uuidReg := regexp.MustCompile(`\t"([0-9a-fA-F\-]{36})"\t`)

	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return &errLoader{err: err}
	}

	return &vtgateLogReaderState{
		logReaderState: logReaderState{
			reader: bufio.NewReader(fd),
			reg:    reg,
			fd:     fd,
		},
		NeedsBindVars: vll.NeedsBindVars,
		uuidReg:       uuidReg,
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

	for {
		line, done, err := s.readLine()
		if err != nil {
			s.fail(fmt.Errorf("error reading file: %w", err))
			return Query{}, false
		}
		if done {
			break
		}

		if len(line) == 0 {
			continue
		}
		line = strings.ReplaceAll(line, "\\n", "")

		match := s.reg.FindStringSubmatch(line)
		if len(match) <= 2 {
			s.fail(fmt.Errorf("line %d: cannot parse log: %s", s.lineNumber, line))
			return Query{}, false
		}

		query := match[1]

		connectionID := s.extractSessionUUIDAsConnectionID(line)
		if connectionID == -1 {
			// something went wrong while extracting the connection ID
			return Query{}, false
		}

		if !s.NeedsBindVars {
			return Query{
				Query:        query,
				Line:         s.lineNumber,
				Type:         SQLQuery,
				ConnectionID: connectionID,
			}, true
		}

		// If we care about bind variables (e.g., running 'trace'), then we parse the query log
		// output into bindVarsVtGate, transform it into something the Vitess library
		// can understand (map[string]*querypb.BindVariable), parse the query string,
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
			Query:        parsedQuery,
			Line:         s.lineNumber,
			Type:         SQLQuery,
			ConnectionID: connectionID,
		}, true
	}

	s.closed = true

	return Query{}, false
}

func (s *vtgateLogReaderState) extractSessionUUIDAsConnectionID(line string) int {
	uuidMatch := s.uuidReg.FindStringSubmatch(line)
	if len(uuidMatch) < 2 {
		s.fail(fmt.Errorf("line %d: cannot extract session UUID: %s", s.lineNumber, line))
		return -1
	}
	sessionUUID := uuidMatch[1]

	// Hash the session UUID using FNV-1a
	h := fnv.New64a()
	_, _ = h.Write([]byte(sessionUUID))
	connectionID := int(h.Sum64())
	return connectionID
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
		case sqltypes.IsIntegral(bvType):
			intVal, ok := value.Value.(float64)
			if !ok {
				return nil, fmt.Errorf("line %d: cannot parse integral bind variable", lineNumber)
			}
			val = strconv.AppendInt(nil, int64(intVal), 10)
		case sqltypes.IsFloat(bvType):
			floatVal, ok := value.Value.(float64)
			if !ok {
				return nil, fmt.Errorf("line %d: cannot parse float bind variable", lineNumber)
			}
			val = strconv.AppendFloat(nil, floatVal, 'f', -1, 64)
		case bvType == sqltypes.Tuple:
			// the query log of vtgate does not list all the values for a tuple
			// instead it lists the following: "v2": {"type": "TUPLE", "value": "2 items"}
			return nil, fmt.Errorf("line %d: cannot parse tuple bind variables", lineNumber)
		}
		if val == nil {
			sval, ok := value.Value.(string)
			if !ok {
				return nil, fmt.Errorf("line %d: cannot parse bind variable value", lineNumber)
			}

			val = []byte(sval)
		}
		bvProcessed[key] = &querypb.BindVariable{
			Type:  bvType,
			Value: val,
		}
	}
	return bvProcessed, nil
}
