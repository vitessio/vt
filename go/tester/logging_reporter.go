/*
Copyright 2025 The Vitess Authors.

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

package tester

import "fmt"

type loggingReporter struct {
	inner Reporter
}

var _ Reporter = (*loggingReporter)(nil)

func (l *loggingReporter) Errorf(format string, args ...interface{}) {
	l.inner.Errorf(format, args...)
}

func (l *loggingReporter) FailNow() {
	l.inner.FailNow()
}

func (l *loggingReporter) Helper() {
	l.inner.Helper()
}

func (l *loggingReporter) AddTestCase(query string, lineNo int) {
	fmt.Printf("Running query %q on line #%d\n", query, lineNo)
	l.inner.AddTestCase(query, lineNo)
}

func (l *loggingReporter) EndTestCase() {
	l.inner.EndTestCase()
}

func (l *loggingReporter) AddFailure(err error) {
	fmt.Printf("Failure added: %s\n", err.Error())
	l.inner.AddFailure(err)
}

func (l *loggingReporter) AddInfo(info string) {
	fmt.Printf("Info: %s\n", info)
	l.inner.AddInfo(info)
}

func (l *loggingReporter) Report() string {
	return l.inner.Report()
}

func (l *loggingReporter) Failed() bool {
	return l.inner.Failed()
}
