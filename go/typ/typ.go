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

package typ

import "strings"

type CmdType int

const (
	Query CmdType = iota
	Error
	RemoveFile
	Skip
	Unknown
	Comment
	CommentWithCommand
	EmptyLine
	SkipIfBelowVersion
	VExplain
	WaitForAuthoritative
)

var commandMap = map[string]CmdType{
	"query":                 Query,
	"error":                 Error,
	"remove_file":           RemoveFile,
	"skip":                  Skip,
	"skip_if_below_version": SkipIfBelowVersion,
	"vexplain":              VExplain,
	"wait_authoritative":    WaitForAuthoritative,
}

func (cmd CmdType) String() string {
	for s, cmdType := range commandMap {
		if cmdType == cmd {
			return s
		}
	}
	return "Unknown command type"
}

func FindType(cmdName string) CmdType {
	key := strings.ToLower(cmdName)
	if v, ok := commandMap[key]; ok {
		return v
	}

	return -1
}
