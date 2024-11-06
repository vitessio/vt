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
	"os"
	"regexp"
)

func ParseMySQLQueryLog(fileName string) (queries []Query, err error) {
	reg := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z)\s+(\d+)\s+(\w+)\s+(.*)`)

	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	// Create a new scanner for the file
	scanner := bufio.NewScanner(fd)

	// Go over each line
	prevQuery := ""
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		// Check if the line matches the pattern
		matches := reg.FindStringSubmatch(line)
		if len(matches) != 5 {
			// In the beginning of the file, we'd have some lines
			// that don't match the regexp, but are not part of the queries.
			// To ignore them, we just check if we have already started with a query.
			if prevQuery != "" {
				prevQuery += line
			}
			continue
		}
		if prevQuery != "" {
			queries = append(queries, Query{Query: prevQuery})
			prevQuery = ""
		}
		if matches[3] == "Query" {
			prevQuery = matches[4]
		}
	}

	// Check for any errors that occurred during the scan
	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return
}
