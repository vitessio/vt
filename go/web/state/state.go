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

package state

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type State struct {
	mu sync.Mutex

	port    int64
	started bool
}

func NewState(port int64) *State {
	return &State{
		port: port,
	}
}

func (s *State) SetStarted(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started = v
}

func (s *State) Started() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *State) GetPort() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

func (s *State) WaitUntilAvailable(timeout time.Duration) bool {
	for {
		select {
		case <-time.After(timeout):
			return false
		case <-time.After(100 * time.Millisecond):
			r, err := http.Get(fmt.Sprintf("http://localhost:%d/", s.port))
			if err != nil {
				continue
			}
			if r.StatusCode == http.StatusOK {
				return true
			}
		}
	}
}
