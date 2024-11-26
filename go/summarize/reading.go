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

package summarize

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/vitessio/vt/go/keys"
)

func readTraceFile(fi fileInfo) (readingSummary, error) {
	switch fi.fileType {
	case traceFile:
		return readTracedQueryFile(fi.filename), nil
	case keysFile:
		return readAnalysedQueryFile(fi.filename), nil
	default:
		return readingSummary{}, errors.New("unknown file format")
	}
}

func getDecoderAndDelim(file *os.File) (*json.Decoder, json.Delim) {
	// Create a decoder
	decoder := json.NewDecoder(file)

	// Read the opening bracket
	val, err := decoder.Token()
	if err != nil {
		exit("Error reading json: " + err.Error())
	}
	delim, ok := val.(json.Delim)
	if !ok {
		exit("Error reading json: expected delimiter")
	}

	// Reset the file pointer to the beginning
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		exit("Error rewinding file: " + err.Error())
	}
	decoder = json.NewDecoder(file)
	return decoder, delim
}

func readTracedQueryFile(fileName string) readingSummary {
	c, err := os.ReadFile(fileName)
	if err != nil {
		exit("Error opening file: " + err.Error())
	}

	type traceOutput struct {
		FileType string        `json:"fileType"`
		Queries  []TracedQuery `json:"queries"`
	}
	var to traceOutput
	err = json.Unmarshal(c, &to)
	if err != nil {
		exit("Error parsing json: " + err.Error())
	}

	sort.Slice(to.Queries, func(i, j int) bool {
		a, err := strconv.Atoi(to.Queries[i].LineNumber)
		if err != nil {
			return false
		}
		b, err := strconv.Atoi(to.Queries[j].LineNumber)
		if err != nil {
			return false
		}
		return a < b
	})

	return readingSummary{
		Name:          fileName,
		TracedQueries: to.Queries,
	}
}

func readAnalysedQueryFile(fileName string) readingSummary {
	c, err := os.ReadFile(fileName)
	if err != nil {
		exit("Error opening file: " + err.Error())
	}

	var ko keys.Output
	err = json.Unmarshal(c, &ko)
	if err != nil {
		exit("Error parsing json: " + err.Error())
	}

	return readingSummary{
		Name:            fileName,
		AnalysedQueries: &ko,
	}
}
