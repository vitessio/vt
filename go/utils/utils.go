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

package utils

import (
	"bytes"
	"fmt"
	"html/template"
)

func GetFuncMap() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"divide": func(a, b any) float64 {
			if b == 0 || b == nil {
				return 0 // Handle division by zero or nil
			}

			// Convert `a` and `b` to float64
			var aFloat, bFloat float64

			switch v := a.(type) {
			case int:
				aFloat = float64(v)
			case float64:
				aFloat = v
			default:
				return 0 // Invalid type, return 0
			}

			switch v := b.(type) {
			case int:
				bFloat = float64(v)
			case float64:
				bFloat = v
			default:
				return 0 // Invalid type, return 0
			}

			return aFloat / bFloat
		},
	}
}

func RenderFile(tplFileName, layoutFileName string, data any) (*bytes.Buffer, error) {
	tmpl := template.Must(template.New(tplFileName).Funcs(GetFuncMap()).ParseFiles(
		"go/web/templates/footer.html",
		"go/web/templates/header.html",
		fmt.Sprintf("go/web/templates/%s", tplFileName),
		fmt.Sprintf("go/web/templates/%s", layoutFileName),
	))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, layoutFileName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %v", err)
	}
	return &buf, nil
}
