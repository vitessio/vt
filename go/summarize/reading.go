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
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/vitessio/vt/go/keys"
)

func readTraceFile(fileName string) (readingSummary, error) {
	// Open the JSON file
	file, err := os.Open(fileName)
	if err != nil {
		return readingSummary{}, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	decoder, val := getDecoderAndDelim(file)

	// Determine the type based on the first delimiter of the JSON file
	switch val {
	case json.Delim('['):
		return readTracedQueryFile(decoder, fileName), nil
	case json.Delim('{'):
		return readAnalysedQueryFile(decoder, fileName), nil
	}

	return readingSummary{}, errors.New("unknown file format")
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

func readTracedQueryFile(decoder *json.Decoder, fileName string) readingSummary {
	var tracedQueries []TracedQuery
	err := decoder.Decode(&tracedQueries)
	if err != nil {
		exit("Error reading json: " + err.Error())
	}

	sort.Slice(tracedQueries, func(i, j int) bool {
		a, err := strconv.Atoi(tracedQueries[i].LineNumber)
		if err != nil {
			return false
		}
		b, err := strconv.Atoi(tracedQueries[j].LineNumber)
		if err != nil {
			return false
		}
		return a < b
	})

	return readingSummary{
		Name:          fileName,
		TracedQueries: tracedQueries,
	}
}

func readAnalysedQueryFile(decoder *json.Decoder, fileName string) readingSummary {
	var output keys.Output
	err := decoder.Decode(&output)
	if err != nil {
		exit("Error reading json: " + err.Error())
	}

	return readingSummary{
		Name:            fileName,
		AnalysedQueries: &output,
	}
}
