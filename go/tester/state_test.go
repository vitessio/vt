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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestState_SetAndCheckStates(t *testing.T) {
	s := &State{}

	// Test setReferenceNext
	require.NoError(t, s.setReference())
	assert.True(t, s.isReferenceSet())
	assert.True(t, s.checkAndClearReference())
	assert.False(t, s.isReferenceSet())

	// Test setSkipNext
	require.NoError(t, s.setSkipNext(), "setSkipNext should not fail")
	assert.True(t, s.isSkipNextSet(), "isSkipNext should return true after setSkipNext")
	assert.True(t, s.checkAndClearSkipNext(), "isSkipNext should return true after setSkipNext")
	assert.False(t, s.isSkipNextSet(), "isSkipNext should return false on second call")

	// Test beginVitessOnly and endVitessOnly
	require.NoError(t, s.beginVitessOnly(), "beginVitessOnly should not fail")
	assert.True(t, s.isVitessOnlySet(), "isVitessOnly should return true after beginVitessOnly")
	require.NoError(t, s.endVitessOnly(), "endVitessOnly should not fail")
	assert.False(t, s.isVitessOnlySet(), "isVitessOnly should return false after endVitessOnly")

	// Test beginMySQLOnly and endMySQLOnly
	require.NoError(t, s.beginMySQLOnly(), "beginMySQLOnly should not fail")
	assert.True(t, s.isMySQLOnlySet(), "isMySQLOnly should return true after beginMySQLOnly")
	require.NoError(t, s.endMySQLOnly(), "endMySQLOnly should not fail")
	assert.False(t, s.isMySQLOnlySet(), "isMySQLOnly should return false after endMySQLOnly")
}

func TestState_StateMutualExclusion(t *testing.T) {
	s := &State{}

	// Set initial state
	require.NoError(t, s.setReference(), "setReferenceNext should not fail")

	// Try to set other states
	assert.Error(t, s.setSkipNext(), "setSkipNext should fail when Reference state is active")
	assert.Error(t, s.beginVitessOnly(), "beginVitessOnly should fail when Reference state is active")
	assert.Error(t, s.beginMySQLOnly(), "beginMySQLOnly should fail when Reference state is active")

	// Clear state and try again
	s.checkAndClearReference()

	assert.NoError(t, s.setSkipNext(), "setSkipNext should not fail after clearing state")
}

func TestState_NormalExecution(t *testing.T) {
	s := &State{}
	assert.True(t, s.normalExecution(), "normalExecution should return true when state is None")

	s.setReference()
	assert.False(t, s.normalExecution(), "normalExecution should return false when state is not None")

	s.checkAndClearReference() // Clear the state
	assert.True(t, s.normalExecution(), "normalExecution should return true after clearing state")
}

func TestState_RunOnVitessAndMySQL(t *testing.T) {
	s := &State{}

	assert.True(t, s.runOnVitess(), "runOnVitess should return true when state is None")
	assert.True(t, s.runOnMySQL(), "runOnMySQL should return true when state is None")

	s.beginVitessOnly()
	assert.True(t, s.runOnVitess(), "runOnVitess should return true when state is VitessOnly")
	assert.False(t, s.runOnMySQL(), "runOnMySQL should return false when state is VitessOnly")

	s.endVitessOnly()
	s.beginMySQLOnly()
	assert.False(t, s.runOnVitess(), "runOnVitess should return false when state is MySQLOnly")
	assert.True(t, s.runOnMySQL(), "runOnMySQL should return true when state is MySQLOnly")
}

func TestState_ShouldSkip(t *testing.T) {
	s := &State{}

	assert.False(t, s.shouldSkip(), "shouldSkip should return false when state is None")

	s.setSkipNext()
	assert.True(t, s.shouldSkip(), "shouldSkip should return true after setSkipNext")
	assert.False(t, s.shouldSkip(), "shouldSkip should return false on second call after setSkipNext")

	checkBinaryIsAtLeast = func(int, string) bool {
		return true
	}
	s.setSkipBelowVersion("testBinary", 1000)
	assert.False(t, s.shouldSkip(), "shouldSkip should return false if binary is at least the required version")
	assert.False(t, s.shouldSkip(), "shouldSkip should always return false here")

	checkBinaryIsAtLeast = func(int, string) bool {
		return false
	}
	s.setSkipBelowVersion("testBinary", 1000)
	assert.True(t, s.shouldSkip(), "shouldSkip should return true after setSkipBelowVersion")
	assert.False(t, s.shouldSkip(), "shouldSkip should always return false here")
}
