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
	"os"
)

type fileType int

const (
	unknownFile fileType = iota
	traceFile
	keysFile
	dbInfoFile
)

var fileTypeMap = map[string]fileType{ //nolint:gochecknoglobals // this is instead of a const
	"trace":  traceFile,
	"keys":   keysFile,
	"dbinfo": dbInfoFile,
}

// getFileType reads the first key-value pair from a JSON file and returns the type of the file
// Note:
func getFileType(filename string) (fileType, error) {
	file, err := os.Open(filename)
	if err != nil {
		return unknownFile, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	token, err := decoder.Token()
	if err != nil {
		return unknownFile, fmt.Errorf("error reading token: %v", err)
	}

	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return unknownFile, errors.New("expected start of object '{'")
	}

	if !decoder.More() {
		return unknownFile, nil
	}

	keyToken, err := decoder.Token()
	if err != nil {
		return unknownFile, fmt.Errorf("error reading key token: %v", err)
	}

	key, ok := keyToken.(string)
	if !ok {
		return unknownFile, fmt.Errorf("expected key to be a string: %s", keyToken)
	}

	valueToken, err := decoder.Token()
	if err != nil {
		return unknownFile, fmt.Errorf("error reading value token: %v", err)
	}

	if key == "fileType" {
		if fileType, ok := fileTypeMap[valueToken.(string)]; ok {
			return fileType, nil
		}
		return unknownFile, fmt.Errorf("unknown FileType: %s", valueToken)
	}

	return unknownFile, nil
}