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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type SQLScriptLoader struct{}

func readData(url string) ([]byte, error) {
	if strings.HasPrefix(url, "http") {
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
	return os.ReadFile(url)
}

func (SQLScriptLoader) Load(url string) ([]Query, error) {
	data, err := readData(url)
	if err != nil {
		return nil, err
	}
	seps := bytes.Split(data, []byte("\n"))
	queries := make([]Query, 0, len(seps))
	newStmt := true
	for i, v := range seps {
		v := bytes.TrimSpace(v)
		s := string(v)
		// Skip comments and empty lines
		switch {
		case strings.HasPrefix(s, "#"):
			newStmt = true
			continue
		case strings.HasPrefix(s, "--"):
			queries = append(queries, Query{Query: s, Line: i + 1})
			newStmt = true
			continue
		case len(s) == 0:
			continue
		}

		if newStmt {
			queries = append(queries, Query{Query: s, Line: i + 1})
		} else {
			lastQuery := queries[len(queries)-1]
			lastQuery.Query = fmt.Sprintf("%s\n%s", lastQuery.Query, s)
			queries[len(queries)-1] = lastQuery
		}

		// Treat new line as a new statement if line ends with ';'
		newStmt = strings.HasSuffix(s, ";")
	}

	// Process queries directly without calling ParseQueries
	finalQueries := make([]Query, 0, len(queries))
	for _, rs := range queries {
		q, err := parseQuery(rs)
		if err != nil {
			return nil, err
		}
		if q != nil {
			finalQueries = append(finalQueries, *q)
		}
	}
	return finalQueries, nil
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
