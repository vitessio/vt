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
	StateErrorExpected
)

type State struct {
	state uint8

	skipBinary  string
	skipVersion int
}

func (s *State) getStateName() string {
	switch s.state {
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
	case StateSkipBelowVersion:
		return "SkipBelowVersion"
	case StateErrorExpected:
		return "ErrorExpected"
	default:
		return "Unknown"
	}
}

func (s *State) setSkipNext() error {
	if s.state != StateNone {
		return fmt.Errorf("cannot set skip next: %s state is already active", s.getStateName())
	}
	s.state = StateSkipNext
	return nil
}

func (s *State) isSkipNextSet() bool {
	return s.state == StateSkipNext
}

func (s *State) checkAndClearSkipNext() bool {
	isSkip := s.state == StateSkipNext
	if isSkip {
		s.state = StateNone
	}
	return isSkip
}

func (s *State) setErrorExpected() error {
	if s.state != StateNone {
		return fmt.Errorf("cannot set error expected: %s state is already active", s.getStateName())
	}
	s.state = StateErrorExpected
	return nil
}

func (s *State) isErrorExpectedSet() bool {
	return s.state == StateErrorExpected
}

func (s *State) checkAndClearErrorExpected() bool {
	isErr := s.state == StateErrorExpected
	if isErr {
		s.state = StateNone
	}
	return isErr
}

func (s *State) setSkipBelowVersion(binary string, version int) error {
	if s.state != StateNone {
		return fmt.Errorf("cannot set skip below version: %s state is already active", s.getStateName())
	}
	s.state = StateSkipBelowVersion
	s.skipBinary = binary
	s.skipVersion = version
	return nil
}

func (s *State) setReference() error {
	if s.state != StateNone {
		return fmt.Errorf("cannot set reference: %s state is already active", s.getStateName())
	}
	s.state = StateReference
	return nil
}

func (s *State) isReferenceSet() bool {
	return s.state == StateReference
}

func (s *State) checkAndClearReference() bool {
	isRef := s.state == StateReference
	if isRef {
		s.state = StateNone
	}
	return isRef
}

func (s *State) beginVitessOnly() error {
	if s.state != StateNone {
		return fmt.Errorf("cannot begin vitess_only: %s state is already active", s.getStateName())
	}
	s.state = StateVitessOnly
	return nil
}

func (s *State) endVitessOnly() error {
	if s.state != StateVitessOnly {
		return fmt.Errorf("cannot end vitess_only: current state is %s", s.getStateName())
	}
	s.state = StateNone
	return nil
}

func (s *State) isVitessOnlySet() bool {
	return s.state == StateVitessOnly
}

func (s *State) beginMySQLOnly() error {
	if s.state != StateNone {
		return fmt.Errorf("cannot begin mysql_only: %s state is already active", s.getStateName())
	}
	s.state = StateMySQLOnly
	return nil
}

func (s *State) endMySQLOnly() error {
	if s.state != StateMySQLOnly {
		return fmt.Errorf("cannot end mysql_only: current state is %s", s.getStateName())
	}
	s.state = StateNone
	return nil
}

func (s *State) isMySQLOnlySet() bool {
	return s.state == StateMySQLOnly
}

func (s *State) normalExecution() bool {
	return s.state == StateNone
}

func (s *State) shouldSkip() bool {
	if s.checkAndClearSkipNext() {
		return true
	}

	if s.state != StateSkipBelowVersion {
		return false
	}

	if s.skipBinary == "" {
		panic("skip below version state is active but skip binary is empty")
	}

	okayToRun := checkBinaryIsAtLeast(s.skipVersion, s.skipBinary)
	s.skipBinary = ""
	s.state = StateNone
	return !okayToRun
}

func (s *State) runOnVitess() bool {
	return s.state == StateNone || s.state == StateVitessOnly
}

func (s *State) runOnMySQL() bool {
	return s.state == StateNone || s.state == StateMySQLOnly
}
