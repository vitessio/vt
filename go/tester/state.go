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

package tester

import (
	"fmt"
	"vitess.io/vitess/go/test/endtoend/utils"
)

var checkBinaryIsAtLeast func(majorVersion int, binary string) bool = utils.BinaryIsAtLeastAtVersion

const (
	StateNone     uint8 = 0
	StateSkipNext uint8 = 1 << iota
	StateSkipBelowVersion
	StateReference
	StateVitessOnly
	StateMySQLOnly
)

type State struct {
	state uint8

	skipBinary  string
	skipVersion int
}

func (ts *State) getStateName() string {
	switch ts.state {
	case StateNone:
		return "None"
	case StateSkipNext:
		return "SkipNext"
	case StateReference:
		return "Reference"
	case StateVitessOnly:
		return "VitessOnly"
	case StateMySQLOnly:
		return "MySQLOnly"
	default:
		return "Unknown"
	}
}

func (ts *State) setSkipBelowVersion(binary string, version int) error {
	if ts.state != StateNone {
		return fmt.Errorf("cannot set skip below version: %s state is already active", ts.getStateName())
	}
	ts.state = StateSkipBelowVersion
	ts.skipBinary = binary
	ts.skipVersion = version
	return nil
}

func (ts *State) setReferenceNext() error {
	if ts.state != StateNone {
		return fmt.Errorf("cannot set reference next: %s state is already active", ts.getStateName())
	}
	ts.state = StateReference
	return nil
}

func (ts *State) isReference() bool {
	isRef := ts.state == StateReference
	if isRef {
		ts.state = StateNone
	}
	return isRef
}

func (ts *State) setSkipNext() error {
	if ts.state != StateNone {
		return fmt.Errorf("cannot set skip next: %s state is already active", ts.getStateName())
	}
	ts.state = StateSkipNext
	return nil
}

func (ts *State) isSkipNext() bool {
	isSkip := ts.state == StateSkipNext
	if isSkip {
		ts.state = StateNone
	}
	return isSkip
}

func (ts *State) beginVitessOnly() error {
	if ts.state != StateNone {
		return fmt.Errorf("cannot begin vitess_only: %s state is already active", ts.getStateName())
	}
	ts.state = StateVitessOnly
	return nil
}

func (ts *State) endVitessOnly() error {
	if ts.state != StateVitessOnly {
		return fmt.Errorf("cannot end vitess_only: current state is %s", ts.getStateName())
	}
	ts.state = StateNone
	return nil
}

func (ts *State) beginMySQLOnly() error {
	if ts.state != StateNone {
		return fmt.Errorf("cannot begin mysql_only: %s state is already active", ts.getStateName())
	}
	ts.state = StateMySQLOnly
	return nil
}

func (ts *State) endMySQLOnly() error {
	if ts.state != StateMySQLOnly {
		return fmt.Errorf("cannot end mysql_only: current state is %s", ts.getStateName())
	}
	ts.state = StateNone
	return nil
}

func (ts *State) isVitessOnly() bool {
	return ts.state == StateVitessOnly
}

func (ts *State) isMySQLOnly() bool {
	return ts.state == StateMySQLOnly
}

func (ts *State) normalExecution() bool {
	return ts.state == StateNone
}

func (state *State) shouldSkip() bool {
	if state.isSkipNext() {
		return true
	}

	if state.state != StateSkipBelowVersion {
		return false
	}

	if state.skipBinary == "" {
		panic("skip below version state is active but skip binary is empty")
	}

	okayToRun := checkBinaryIsAtLeast(state.skipVersion, state.skipBinary)
	state.skipBinary = ""
	state.state = StateNone
	return !okayToRun
}

func (ts *State) runOnVitess() bool {
	return ts.state == StateNone || ts.state == StateVitessOnly
}

func (ts *State) runOnMySQL() bool {
	return ts.state == StateNone || ts.state == StateMySQLOnly
}
