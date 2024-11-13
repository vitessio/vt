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
	"strings"
)

type SQLScriptLoader struct{}

type sqlLogReaderState struct {
	logReaderState
}

func (s *sqlLogReaderState) Next() (Query, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.err != nil {
		return Query{}, false
	}

	var currentQuery Query
	newStmt := true
	for s.scanner.Scan() {
		s.lineNumber++
		line := s.scanner.Text()
		line = strings.TrimSpace(line)

		switch {
		case len(line) == 0:
			continue
		case strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "--"):
			newStmt = true
			q := Query{Query: line, Line: s.lineNumber}
			pq, err := parseQuery(q)
			if err != nil {
				s.err = err
				return Query{}, false
			}
			if pq == nil {
				continue
			}
			return *pq, true
		}

		if newStmt {
			currentQuery = Query{Query: line, Line: s.lineNumber}
		} else {
			currentQuery.Query = fmt.Sprintf("%s\n%s", currentQuery.Query, line)
		}

		// Treat new line as a new statement if line ends with ';'
		newStmt = strings.HasSuffix(line, ";")
		if newStmt {
			pq, err := parseQuery(currentQuery)
			if err != nil {
				s.err = err
				return Query{}, false
			}
			if pq == nil {
				continue
			}
			return *pq, true
		}
	}
	if !newStmt {
		s.err = errors.New("EOF: missing semicolon")
	}
	return Query{}, false
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

func (SQLScriptLoader) Loadit(filename string) IteratorLoader {
	var scanner *bufio.Scanner
	var fd *os.File

	if strings.HasPrefix(filename, "http") {
		data, err := readData(filename)
		if err != nil {
			return &errLoader{err: err}
		}
		scanner = bufio.NewScanner(bytes.NewReader(data))
	} else {
		var err error
		fd, err = os.OpenFile(filename, os.O_RDONLY, 0)
		if err != nil {
			return &errLoader{err: err}
		}
		scanner = bufio.NewScanner(fd)
	}

	return &sqlLogReaderState{
		logReaderState: logReaderState{
			scanner: scanner,
			fd:      fd,
		},
	}
}

func (SQLScriptLoader) Load(url string) ([]Query, error) {
	loader := SQLScriptLoader{}.Loadit(url)
	return makeSlice(loader)
}

// Helper function to parse individual queries
func parseQuery(rs Query) (*Query, error) {
	realS := rs.Query
	s := rs.Query
	q := Query{Line: rs.Line, Type: Unknown}

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
