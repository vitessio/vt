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
	"os"
	"regexp"
	"sync"
)

type (
	MySQLLogLoader struct{}

	logReaderState struct {
		fd         *os.File
		scanner    *bufio.Scanner
		reg        *regexp.Regexp
		mu         sync.Mutex
		lineNumber int
		closed     bool
		err        error
	}

	mysqlLogReaderState struct {
		logReaderState
		prevQuery  string
		queryStart int
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

	for s.scanner.Scan() {
		s.lineNumber++
		line := s.scanner.Text()
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
			query := Query{
				Query: s.prevQuery,
				Line:  s.queryStart,
				Type:  QueryT,
			}
			s.prevQuery = ""

			// If the new line is a query, store it for next iteration
			if matches[3] == "Query" {
				s.prevQuery = matches[4]
				s.queryStart = s.lineNumber
			}

			return query, true
		}

		// Start a new query if this line is a query
		if matches[3] == "Query" {
			s.prevQuery = matches[4]
			s.queryStart = s.lineNumber
		}
	}
	s.closed = true

	// Return the last query if we have one
	if s.prevQuery != "" {
		query := Query{
			Query: s.prevQuery,
			Line:  s.queryStart,
			Type:  QueryT,
		}
		s.prevQuery = ""
		return query, true
	}

	s.err = s.scanner.Err()
	return Query{}, false
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

func (s *logReaderState) NextLine() (string, bool) {
	more := s.scanner.Scan()
	if !more {
		return "", false
	}

	return s.scanner.Text(), true
}

func (MySQLLogLoader) Load(fileName string) IteratorLoader {
	reg := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z)\s+(\d+)\s+(\w+)\s+(.*)`)

	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return &errLoader{err}
	}

	scanner := bufio.NewScanner(fd)

	return &mysqlLogReaderState{
		logReaderState: logReaderState{
			scanner: scanner,
			reg:     reg,
			fd:      fd,
		},
	}
}
