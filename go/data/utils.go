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
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type FileType int

const (
	UnknownFile FileType = iota
	TraceFile
	KeysFile
	DBInfoFile
	TransactionFile
)

var fileTypeMap = map[string]FileType{ //nolint:gochecknoglobals // this is instead of a const
	"trace":        TraceFile,
	"keys":         KeysFile,
	"dbinfo":       DBInfoFile,
	"transactions": TransactionFile,
}

// GetFileType reads the first key-value pair from a JSON file and returns the type of the file
// Note:
func GetFileType(filename string) (FileType, error) {
	file, err := os.Open(filename)
	if err != nil {
		return UnknownFile, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	token, err := decoder.Token()
	if err != nil {
		return UnknownFile, fmt.Errorf("error reading json token: %v", err)
	}

	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return UnknownFile, errors.New("expected start of object '{'")
	}

	if !decoder.More() {
		return UnknownFile, nil
	}

	keyToken, err := decoder.Token()
	if err != nil {
		return UnknownFile, fmt.Errorf("error reading json key token: %v", err)
	}

	key, ok := keyToken.(string)
	if !ok {
		return UnknownFile, fmt.Errorf("expected key to be a string: %s", keyToken)
	}

	valueToken, err := decoder.Token()
	if err != nil {
		return UnknownFile, fmt.Errorf("error reading json value token: %v", err)
	}

	if key == "fileType" {
		s, ok := valueToken.(string)
		if !ok {
			return UnknownFile, fmt.Errorf("expected value to be a string: %s", valueToken)
		}
		if fileType, ok := fileTypeMap[s]; ok {
			return fileType, nil
		}
		return UnknownFile, fmt.Errorf("unknown FileType: %s", valueToken)
	}

	return UnknownFile, nil
}
