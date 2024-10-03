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

╭─systay@Andress-Mac-Studio.local ~/dev/vitess-tester  ‹main*›
╰─➤  go build -o vt2 ./go/                                                                                                                                                                                                                                                                                                                                              1 ↵
no Go files in /Users/systay/dev/vitess-tester/go
╭─systay@Andress-Mac-Studio.local ~/dev/vitess-tester  ‹main*›
╰─➤  go build -o vt2 ./go/.                                                                                                                                                                                                                                                                                                                                             1 ↵
no Go files in /Users/systay/dev/vitess-tester/go
╭─systay@Andress-Mac-Studio.local ~/dev/vitess-tester  ‹main*›
╰─➤  go build -o vt2 ./go/..                                                                                                                                                                                                                                                                                                                                            1 ↵
no Go files in /Users/systay/dev/vitess-tester
╭─systay@Andress-Mac-Studio.local ~/dev/vitess-tester  ‹main*›
╰─➤  go build -o vt2 ./go/...                                                                                                                                                                                                                                                                                                                                           1 ↵
go: cannot write multiple packages to non-directory vt2


*/

package main

import (
	"github.com/vitessio/vitess-tester/go/cmd"
)

func main() {
	cmd.Execute()
}
