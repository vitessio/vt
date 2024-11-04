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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTable(t *testing.T) {
	tests := []struct {
		headers  []string
		rows     [][]string
		expected string
	}{
		{
			headers: []string{"a", "b"},
			rows:    [][]string{{"1", "2"}, {"3", "4"}},
			expected: `|a|b|
|---|---|
|1|2|
|3|4|
`,
		},
		{
			headers: []string{"header1", "header2", "header3"},
			rows:    [][]string{{"val1", "val2", "val3"}},
			expected: `|header1|header2|header3|
|---|---|---|
|val1|val2|val3|
`,
		},
		{
			headers: []string{"x"},
			rows:    [][]string{{"1"}, {"2"}},
			expected: `|x|
|---|
|1|
|2|
`,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("headers: %v, rows: %v", tt.headers, tt.rows), func(t *testing.T) {
			md := &markDown{}
			md.PrintTable(tt.headers, tt.rows)
			assert.Equal(t, tt.expected, md.String())
		})
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "| alla * kan _ älska !",
			expected: `&#124; alla \* kan \_ älska \!`,
		},
		{
			input:    "[hello](world)",
			expected: `\[hello\]\(world\)`,
		},
		{
			input:    "# This is a test + example",
			expected: `\# This is a test \+ example`,
		},
		{
			input:    "simple text",
			expected: "simple text",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input: %v", tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, markdownEscaper.Replace(tt.input))
		})
	}
}

func TestPrintHeader(t *testing.T) {
	tests := []struct {
		input    string
		level    int
		expected string
	}{
		{
			input:    "Header 1",
			level:    1,
			expected: "# Header 1\n",
		},
		{
			input:    "Section",
			level:    2,
			expected: "## Section\n",
		},
		{
			input:    "Subsection",
			level:    3,
			expected: "### Subsection\n",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("header: %s, level: %d", tt.input, tt.level), func(t *testing.T) {
			md := &markDown{}
			md.PrintHeader(tt.input, tt.level)
			assert.Equal(t, tt.expected, md.String())
		})
	}
}

func TestPrintln(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Hello, world!",
			expected: "Hello, world\\!\n",
		},
		{
			input:    "This | should * be _ escaped!",
			expected: "This &#124; should \\* be \\_ escaped\\!\n",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input: %s", tt.input), func(t *testing.T) {
			md := &markDown{}
			md.Println(tt.input)
			assert.Equal(t, tt.expected, md.String())
		})
	}
}

func TestPrintf(t *testing.T) {
	tests := []struct {
		format   string
		args     []any
		expected string
	}{
		{
			format:   "Hello, %s!",
			args:     []any{"world"},
			expected: "Hello, world\\!",
		},
		{
			format:   "This | is a number: %d",
			args:     []any{42},
			expected: "This &#124; is a number: 42",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("format: %s", tt.format), func(t *testing.T) {
			md := &markDown{}
			md.Printf(tt.format, tt.args...)
			assert.Equal(t, tt.expected, md.String())
		})
	}
}
