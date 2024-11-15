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
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type SlowQueryLogLoader struct{}

type slowQueryLogReaderState struct {
	logReaderState
}

type lineProcessorState struct {
	currentQuery     Query
	newStmt          bool
	hasQueryMetadata bool
}

func (s *slowQueryLogReaderState) Next() (Query, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.err != nil {
		return Query{}, false
	}

	state := &lineProcessorState{
		newStmt: true,
	}

	for {
		line, done, err := s.readLine()
		if err != nil {
			s.err = fmt.Errorf("error reading file: %w", err)
			return Query{}, false
		}
		if done {
			break
		}
		s.lineNumber++
		line = strings.TrimSpace(line)

		result, done, err := s.processLine(line, state)
		if err != nil {
			s.err = err
			return Query{}, false
		}
		if done {
			return result, true
		}
	}

	if !state.newStmt && state.currentQuery.Query != "" {
		s.err = errors.New("EOF: missing semicolon")
	}
	return Query{}, false
}

func (s *slowQueryLogReaderState) processLine(line string, state *lineProcessorState) (Query, bool, error) {
	switch {
	case len(line) == 0:
		return Query{}, false, nil
	case strings.HasPrefix(line, "#"):
		hasMetadata, err := s.processCommentLine(line, state)
		if err != nil {
			return Query{}, false, fmt.Errorf("line %d: %w", s.lineNumber, err)
		}
		if hasMetadata {
			state.hasQueryMetadata = true
		}
		return Query{}, false, nil
	case strings.HasPrefix(line, "SET timestamp=") && state.hasQueryMetadata:
		err := s.processSetTimestampLine(line, state)
		if err != nil {
			return Query{}, false, fmt.Errorf("line %d: %w", s.lineNumber, err)
		}
		state.hasQueryMetadata = false
		return Query{}, false, nil
	case strings.HasPrefix(line, "--"):
		pq, err := s.processStatementLine(line, state)
		if err != nil {
			return Query{}, false, fmt.Errorf("line %d: %w", s.lineNumber, err)
		}
		if pq != nil {
			return *pq, true, nil
		}
		return Query{}, false, nil
	case state.newStmt:
		s.startNewQuery(line, state)
	default:
		s.appendToCurrentQuery(line, state)
	}

	state.newStmt = strings.HasSuffix(line, ";")
	if state.newStmt {
		pq, err := s.processEndOfStatement(line, state)
		if err != nil {
			return Query{}, false, fmt.Errorf("line %d: %w", s.lineNumber, err)
		}
		if pq != nil {
			return *pq, true, nil
		}
	}

	return Query{}, false, nil
}

func (s *slowQueryLogReaderState) processCommentLine(line string, state *lineProcessorState) (bool, error) {
	if strings.HasPrefix(line, "# Query_time:") {
		if err := parseQueryMetrics(line, &state.currentQuery); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *slowQueryLogReaderState) processSetTimestampLine(line string, state *lineProcessorState) error {
	tsStr := strings.TrimPrefix(line, "SET timestamp=")
	tsStr = strings.TrimSuffix(tsStr, ";")
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp '%s': %w", tsStr, err)
	}
	state.currentQuery.Timestamp = ts
	return nil
}

func (s *slowQueryLogReaderState) processStatementLine(line string, state *lineProcessorState) (*Query, error) {
	state.newStmt = true
	q := Query{Query: line, Line: s.lineNumber}
	pq, err := parseQuery(q)
	if err != nil {
		return nil, err
	}
	return pq, nil
}

func (s *slowQueryLogReaderState) processEndOfStatement(line string, state *lineProcessorState) (*Query, error) {
	if strings.HasPrefix(line, "SET timestamp=") && state.currentQuery.QueryTime > 0 {
		return nil, nil
	}
	pq, err := parseQuery(state.currentQuery)
	if err != nil {
		return nil, err
	}
	return pq, nil
}

func (s *slowQueryLogReaderState) startNewQuery(line string, state *lineProcessorState) {
	state.currentQuery.Query = line
	state.currentQuery.Line = s.lineNumber
}

func (s *slowQueryLogReaderState) appendToCurrentQuery(line string, state *lineProcessorState) {
	state.currentQuery.Query = fmt.Sprintf("%s\n%s", state.currentQuery.Query, line)
}

// parseQueryMetrics parses the metrics from the comment line and assigns them to the Query struct.
func parseQueryMetrics(line string, q *Query) error {
	line = strings.TrimPrefix(line, "#")
	line = strings.TrimSpace(line)

	fields := strings.Fields(line)

	i := 0
	for i < len(fields) {
		field := fields[i]
		if !strings.HasSuffix(field, ":") {
			return fmt.Errorf("unexpected field format '%s'", field)
		}

		// Remove the trailing colon to get the key
		key := strings.TrimSuffix(field, ":")
		if i+1 >= len(fields) {
			return fmt.Errorf("missing value for key '%s'", key)
		}
		value := fields[i+1]

		// Assign to Query struct based on key
		switch key {
		case "Query_time":
			fval, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid Query_time value '%s'", value)
			}
			q.QueryTime = fval
		case "Lock_time":
			fval, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid Lock_time value '%s'", value)
			}
			q.LockTime = fval
		case "Rows_sent":
			ival, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid Rows_sent value '%s'", value)
			}
			q.RowsSent = ival
		case "Rows_examined":
			ival, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid Rows_examined value '%s'", value)
			}
			q.RowsExamined = ival
		}
		i += 2 // Move to the next key-value pair
	}

	return nil
}

func readData(url string) ([]byte, error) {
	client := http.Client{}
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get data from %s, status code %d", url, res.StatusCode)
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func (SlowQueryLogLoader) Load(filename string) IteratorLoader {
	var reader *bufio.Reader
	var fd *os.File

	if strings.HasPrefix(filename, "http") {
		data, err := readData(filename)
		if err != nil {
			return &errLoader{err: err}
		}
		reader = bufio.NewReader(bytes.NewReader(data))
	} else {
		var err error
		fd, err = os.OpenFile(filename, os.O_RDONLY, 0)
		if err != nil {
			return &errLoader{err: err}
		}
		reader = bufio.NewReader(fd)
	}

	return &slowQueryLogReaderState{
		logReaderState: logReaderState{
			fd:     fd,
			reader: reader,
		},
	}
}

// Helper function to parse individual queries
func parseQuery(rs Query) (*Query, error) {
	realS := rs.Query
	s := rs.Query
	q := Query{
		Line:         rs.Line,
		Type:         Unknown,
		QueryTime:    rs.QueryTime,
		LockTime:     rs.LockTime,
		RowsSent:     rs.RowsSent,
		RowsExamined: rs.RowsExamined,
		Timestamp:    rs.Timestamp,
	}

	if len(s) < 3 {
		return nil, nil
	}

	switch {
	case strings.HasPrefix(s, "#"):
		q.Type = Comment
		return &q, nil
	case strings.HasPrefix(s, "--"):
		q.Type = CommentWithCommand
		if len(s) > 2 && s[2] == ' ' {
			s = s[3:]
		} else {
			s = s[2:]
		}
	case s[0] == '\n':
		q.Type = EmptyLine
		return &q, nil
	}

	i := findFirstWord(s)
	if i > 0 {
		q.FirstWord = s[:i]
	}
	q.Query = s[i:]

	if q.Type == Unknown || q.Type == CommentWithCommand {
		if err := q.getQueryType(realS); err != nil {
			return nil, err
		}
	}

	return &q, nil
}

// findFirstWord calculates the length of the first word in the string
func findFirstWord(s string) int {
	i := 0
	for i < len(s) && s[i] != '(' && s[i] != ' ' && s[i] != ';' && s[i] != '\n' {
		i++
	}
	return i
}
