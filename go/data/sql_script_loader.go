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

	"github.com/vitessio/vt/go/typ"
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
		// we will skip # comment here
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
			lastQuery = Query{Query: fmt.Sprintf("%s\n%s", lastQuery.Query, s), Line: lastQuery.Line}
			queries[len(queries)-1] = lastQuery
		}

		// if the line has a ; in the end, we will treat new line as the new statement.
		newStmt = strings.HasSuffix(s, ";")
	}

	return ParseQueries(queries...)
}

// ParseQueries parses an array of string into an array of Query object.
// Note: a Query statement may reside in several lines.
func ParseQueries(qs ...Query) ([]Query, error) {
	queries := make([]Query, 0, len(qs))
	for _, rs := range qs {
		query, err := parseQuery(rs)
		if err != nil {
			return nil, err
		}
		if query.Type != typ.Unknown {
			queries = append(queries, query)
		}
	}
	return queries, nil
}

// parseQuery parses a single Query object
func parseQuery(rs Query) (Query, error) {
	q := Query{
		Type: typ.Unknown,
		Line: rs.Line,
	}

	// Skip invalid queries
	if len(rs.Query) < 3 {
		return q, nil
	}

	// Parse initial query type and clean query string
	q, s := parseInitialType(rs.Query)

	// Only process further if not a pure comment
	if q.Type != typ.Comment {
		q = parseFirstWord(q, s)

		if q.Type == typ.Unknown || q.Type == typ.CommentWithCommand {
			if err := q.getQueryType(rs.Query); err != nil {
				return Query{}, err
			}
		}
	}

	return q, nil
}

// parseInitialType determines the initial query type based on prefix
func parseInitialType(s string) (Query, string) {
	q := Query{
		Type: typ.Unknown,
	}

	switch {
	case s[0] == '#':
		q.Type = typ.Comment
		return q, s
	case s[0:2] == "--":
		q.Type = typ.CommentWithCommand
		if s[2] == ' ' {
			return q, s[3:]
		}
		return q, s[2:]
	case s[0] == '\n':
		q.Type = typ.EmptyLine
		return q, s
	}

	return q, s
}

// parseFirstWord extracts the first word from the query string
func parseFirstWord(q Query, s string) Query {
	firstWordEnd := findFirstWordEnd(s)

	if firstWordEnd > 0 {
		q.FirstWord = s[:firstWordEnd]
		q.Query = s[firstWordEnd:]
	} else {
		q.Query = s
	}

	return q
}

// findFirstWordEnd finds the index where the first word ends
func findFirstWordEnd(s string) int {
	for i := range len(s) {
		if s[i] == '(' || s[i] == ' ' || s[i] == ';' || s[i] == '\n' {
			return i
		}
	}
	return 0
}
