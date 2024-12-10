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

package markdown

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type MarkDown struct {
	buffer bytes.Buffer
}

func (m *MarkDown) String() string {
	return m.buffer.String()
}

func (m *MarkDown) PrintHeader(s string, lvl int) {
	m.buffer.WriteString(strings.Repeat("#", lvl) + " " + s)
	m.NewLine()
}

func (m *MarkDown) Println(s string) {
	m.buffer.WriteString(s)
	m.NewLine()
}

func (m *MarkDown) NewLine() {
	m.buffer.WriteString("\n")
}

func (m *MarkDown) Printf(format string, args ...any) {
	m.buffer.WriteString(fmt.Sprintf(format, args...))
}

func (m *MarkDown) PrintTable(headers []string, rows [][]string) {
	size := len(headers)
	headerLine := "|" + strings.Join(headers, "|") + "|\n"
	m.buffer.WriteString(headerLine)
	m.buffer.WriteString("|" + strings.Repeat("---|", size) + "\n")
	for _, row := range rows {
		if len(row) != size {
			panic("row size does not match headers")
		}
		rowLine := "|" + strings.Join(row, "|") + "|\n"
		m.buffer.WriteString(rowLine)
	}
	m.NewLine()
}

func (m *MarkDown) Write(p []byte) (n int, err error) {
	return m.buffer.Write(p)
}

func (m *MarkDown) WriteTo(w io.Writer) (n int64, err error) {
	return m.buffer.WriteTo(w)
}
