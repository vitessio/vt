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
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"sync"
)

type (
	MySQLLogLoader struct{}

	logReaderState struct {
		fd         *os.File
		reader     *bufio.Reader
		reg        *regexp.Regexp
		mu         sync.Mutex
		lineNumber int
		closed     bool
		err        error
	}

	mysqlLogReaderState struct {
		logReaderState
		prevQuery        string
		queryStart       int
		prevConnectionID int
	}
)

func makeSlice(loader IteratorLoader) ([]Query, error) {
	var queries []Query
	for {
		query, ok := loader.Next()
		if !ok {
			break
		}
		queries = append(queries, query)
	}

	return queries, loader.Close()
}

func (s *mysqlLogReaderState) Next() (Query, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return Query{}, false
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

		if len(line) == 0 {
			continue
		}

		matches := s.reg.FindStringSubmatch(line)
		if len(matches) != 5 {
			if s.prevQuery != "" {
				s.prevQuery += "\n" + line
			}
			continue
		}

		// If we have a previous query, return it before processing the new line
		if s.prevQuery != "" {
			return s.processQuery(matches), true
		}

		// Start a new query if this line is a query
		if matches[3] == "Query" {
			s.prevQuery = matches[4]
			s.queryStart = s.lineNumber
			connID, err := strconv.Atoi(matches[2])
			if err != nil {
				s.err = fmt.Errorf("invalid connection id at line %d: %w", s.lineNumber, err)
				return Query{}, false
			}
			s.prevConnectionID = connID
		}
	}
	s.closed = true

	// Return the last query if we have one
	if s.prevQuery != "" {
		return s.finalizeQuery(), true
	}

	return Query{}, false
}

func (s *logReaderState) readLine() (string, bool, error) {
	s.lineNumber++
	line, isPrefix, err := s.reader.ReadLine()
	if err == io.EOF {
		return "", true, nil
	}
	if err != nil {
		return "", false, err
	}

	// Handle lines longer than the buffer size
	totalLine := append([]byte{}, line...)
	for isPrefix {
		line, isPrefix, err = s.reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false, err
		}
		totalLine = append(totalLine, line...)
	}
	return string(totalLine), false, nil
}

func (s *mysqlLogReaderState) finalizeQuery() Query {
	query := Query{
		Query:        s.prevQuery,
		Line:         s.queryStart,
		Type:         QueryT,
		ConnectionID: s.prevConnectionID,
	}
	s.prevQuery = ""
	s.prevConnectionID = 0
	return query
}

func (s *mysqlLogReaderState) processQuery(matches []string) Query {
	query := Query{
		Query:        s.prevQuery,
		Line:         s.queryStart,
		Type:         QueryT,
		ConnectionID: s.prevConnectionID,
	}
	s.prevQuery = ""
	s.prevConnectionID = 0

	// If the new line is a query, store it for next iteration
	if matches[3] == "Query" {
		s.prevQuery = matches[4]
		s.queryStart = s.lineNumber
		connID, err := strconv.Atoi(matches[2])
		if err != nil {
			s.err = fmt.Errorf("invalid connection id at line %d: %w", s.lineNumber, err)
			return Query{}
		}
		s.prevConnectionID = connID
	}
	return query
}

func (s *logReaderState) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed && s.fd != nil {
		ferr := s.fd.Close()
		if ferr != nil {
			s.err = errors.Join(s.err, ferr)
		}
		s.closed = true
	}

	return s.err
}

func (MySQLLogLoader) Load(fileName string) IteratorLoader {
	reg := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z)\s+(\d+)\s+(\w+)\s+(.*)`)

	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return &errLoader{err}
	}

	return &mysqlLogReaderState{
		logReaderState: logReaderState{
			reader: bufio.NewReader(fd),
			reg:    reg,
			fd:     fd,
		},
	}
}
