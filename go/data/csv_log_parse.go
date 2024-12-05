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
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type (
	CSVLogLoader struct {
		Config CSVConfig
	}

	CSVConfig struct {
		Header bool

		QueryField        int
		ConnectionIDField int
		QueryTimeField    int
		LockTimeField     int
		RowsSentField     int
		RowsExaminedField int
		TimestampField    int
	}

	csvLogReaderState struct {
		file   *os.File
		reader *csv.Reader

		CSVConfig
	}
)

func NewEmptyCSVConfig(header bool, queryField int) CSVConfig {
	return CSVConfig{
		Header:     header,
		QueryField: queryField,

		ConnectionIDField: -1,
		QueryTimeField:    -1,
		LockTimeField:     -1,
		RowsSentField:     -1,
		RowsExaminedField: -1,
		TimestampField:    -1,
	}
}

func (c CSVLogLoader) Load(fileName string) IteratorLoader {
	fd, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return &errLoader{err}
	}

	reader := csv.NewReader(fd)
	logReader := &csvLogReaderState{
		file:      fd,
		reader:    reader,
		CSVConfig: c.Config,
	}

	if c.Config.Header {
		_, _ = reader.Read()
	}

	return logReader
}

func (c *csvLogReaderState) Next() (Query, bool) {
	record, err := c.reader.Read()
	if err == io.EOF {
		return Query{}, false
	}
	if err != nil {
		panic(err)
	}

	l, _ := c.reader.FieldPos(0)

	recordToInt := func(idx int) int {
		if idx == -1 {
			return 0
		}
		val := record[idx]
		i, err := strconv.Atoi(val)
		if err != nil {
			panic(err)
		}
		return i
	}

	recordToFloat64 := func(idx int) float64 {
		if idx == -1 {
			return 0
		}
		val := record[idx]
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			panic(err)
		}
		return f
	}

	recordToTime := func(idx int) int64 {
		if idx == -1 {
			return 0
		}
		val := record[idx]
		t, err := time.Parse(time.DateTime, val)
		if err != nil {
			panic(err)
		}
		return t.Unix()
	}

	query := record[c.QueryField]
	query = strings.Trim(query, "\"")

	return Query{
		Query:        query,
		Line:         l,
		Type:         SQLQuery,
		UsageCount:   1,
		ConnectionID: recordToInt(c.ConnectionIDField),
		QueryTime:    recordToFloat64(c.QueryTimeField),
		LockTime:     recordToFloat64(c.LockTimeField),
		RowsSent:     recordToInt(c.RowsSentField),
		RowsExamined: recordToInt(c.RowsExaminedField),
		Timestamp:    recordToTime(c.TimestampField),
	}, true
}

func (c *csvLogReaderState) Close() error {
	return c.file.Close()
}
