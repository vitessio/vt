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
	"bytes"
	"fmt"
	"strings"
)

type markDown struct {
	buffer bytes.Buffer
}

var markdownEscaper = strings.NewReplacer( //nolint:gochecknoglobals // this is instead of a constant
	`|`, `&#124;`,
	`*`, `\*`,
	`_`, `\_`,
	`[`, `\[`,
	`]`, `\]`,
	`(`, `\(`,
	`)`, `\)`,
	`#`, `\#`,
	`+`, `\+`,
	`-`, `\-`,
	`!`, `\!`,
	`>`, `\>`,
	`~`, `\~`,
	`\`, `\\`,
)

func escape(s []string) []string {
	for i, v := range s {
		s[i] = markdownEscaper.Replace(v)
	}
	return s
}

func (m *markDown) String() string {
	return m.buffer.String()
}

func (m *markDown) PrintHeader(s string, lvl int) {
	m.buffer.WriteString(strings.Repeat("#", lvl) + " " + markdownEscaper.Replace(s) + "\n")
}

func (m *markDown) Println(s string) {
	m.buffer.WriteString(markdownEscaper.Replace(s) + "\n")
}

func (m *markDown) Printf(format string, args ...any) {
	m.buffer.WriteString(markdownEscaper.Replace(fmt.Sprintf(format, args...)))
}

func (m *markDown) PrintTable(headers []string, rows [][]string) {
	size := len(headers)
	headerLine := "|" + strings.Join(escape(headers), "|") + "|\n"
	m.buffer.WriteString(headerLine)
	m.buffer.WriteString("|" + strings.Repeat("---|", size) + "\n")
	for _, row := range rows {
		if len(row) != size {
			panic("row size does not match headers")
		}
		rowLine := "|" + strings.Join(escape(row), "|") + "|\n"
		m.buffer.WriteString(rowLine)
	}
}
