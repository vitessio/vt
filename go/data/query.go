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
	"fmt"

	log "github.com/sirupsen/logrus"
)

type (
	Loader interface {
		Load(url string) ([]Query, error)
	}

	IteratorLoader interface {
		// Next returns the next query in the log file. The boolean return value is false if there are no more queries.
		Next() (Query, bool)

		// Close closes the iterator. If any errors have been accumulated, they are returned here.
		Close() error
	}

	Query struct {
		FirstWord string
		Query     string
		Line      int
		Type      CmdType
	}
)

// for a single query, it has some prefix. Prefix mapps to a query type.
// e.g query_vertical maps to typ.Q_QUERY_VERTICAL
func (q *Query) getQueryType(qu string) error {
	tp := FindType(q.FirstWord)
	if tp > 0 {
		q.Type = tp
	} else {
		// No mysqltest command matched
		if q.Type != CommentWithCommand {
			// A query that will sent to vitess
			q.Query = qu
			q.Type = QueryT
		} else {
			log.WithFields(log.Fields{"line": q.Line, "command": q.FirstWord, "arguments": q.Query}).Error("invalid command")
			return fmt.Errorf("invalid command %s", q.FirstWord)
		}
	}
	return nil
}
