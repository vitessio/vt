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

var fileTypeMap = map[string]fileType{
	"trace":  traceFile,
	"keys":   keysFile,
	"dbinfo": dbInfoFile,
}

// getFileType reads the first key-value pair from a JSON file and returns the type of the file
// Note:
func getFileType(filename string) (fileType, error) {
	// read json file
	file, err := os.Open(filename)
	if err != nil {
		return unknownFile, errors.New(fmt.Sprintf("error opening file: %v", err))
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	token, err := decoder.Token()
	if err != nil {
		return unknownFile, errors.New(fmt.Sprintf("Error reading token: %v", err))
	}

	// Ensure the token is the start of an object
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return unknownFile, errors.New(fmt.Sprintf("Expected start of object '{'"))
	}

	// Read the key-value pairs
	for decoder.More() {
		// Read the key
		keyToken, err := decoder.Token()
		if err != nil {
			return unknownFile, errors.New(fmt.Sprintf("Error reading key token: %v", err))
		}

		key, ok := keyToken.(string)
		if !ok {
			return unknownFile, errors.New(fmt.Sprintf("Expected key to be a string: %s", keyToken))
		}

		// Read the value
		valueToken, err := decoder.Token()
		if err != nil {
			return unknownFile, errors.New(fmt.Sprintf("Error reading value token: %v", err))
		}

		// Check if the key is "FileType"
		if key == "FileType" {
			if fileType, ok := fileTypeMap[valueToken.(string)]; ok {
				return fileType, nil
			} else {
				return unknownFile, errors.New(fmt.Sprintf("Unknown FileType: %s", valueToken))
			}
		} else {
			// Currently we expect the first key to be FileType, for optimization reasons
			return unknownFile, nil
		}
	}
	return unknownFile, nil
}
