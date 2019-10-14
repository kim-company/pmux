// Copyright 2019 KIM Keep In Mind GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tmux provides an interface for a subset of tmux functions.
package tmux

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/pipe.v2"
)

const defaultCmdExecTimeout = time.Millisecond * 100

// verify returns an error if it is not able to find the tmux executable.
func verify() error {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux is not available: %w", err)
	}
	log.Printf("[INFO] using tmux located at: %v", path)
	return nil
}

// Version returns tmux version. Returns an error only if the command cannot
// be executed, does not check the output produced.
func Version() (string, error) {
	p := pipe.Exec("tmux", "-V")
	v, err := pipe.OutputTimeout(p, defaultCmdExecTimeout)
	if err != nil {
		return "", fmt.Errorf("unable to fetch tmux version: %w", err)
	}
	return string(v), nil
}

func NewSID() string {
	return "pmux-" + uuid.New().String()
}

func validateSID(s string) error {
	if !strings.HasPrefix(s, "pmux-") {
		return fmt.Errorf("session identifier %v does not belong to pmux", s)
	}
	return nil
}

// NewSession creates a new tmux session using "name" as the name of the executable
// to be started, and "sid" as tmux session identifier. "sid" will be validated using
// the `validateSID` function, and the function will return an error if the validation
// does not pass. Use `NewSID` to build a valid session identifier, or validate it first
// manually.
// Note that there are not guarantees that the session will still be running after
// this function returns.
func NewSession(sid, name string, args ...string) error {
	if err := validateSID(sid); err != nil {
		return fmt.Errorf("unable to create new tmux session: %w", err)
	}
	args = append([]string{"new", "-s", sid, "-d", name}, args...)
	p := pipe.Exec("tmux", args...)
	if err := pipe.RunTimeout(p, defaultCmdExecTimeout); err != nil {
		return fmt.Errorf("unable to create new tmux session: %w", err)
	}
	return nil
}

// KillSession destroys a session, terminating all its child processes. If the session
// identifier does not belong to pmux returns an error.
func KillSession(sid string) error {
	if err := validateSID(sid); err != nil {
		return fmt.Errorf("cannot terminate session: %w", err)
	}
	p := pipe.Exec("tmux", "kill-session", "-t", sid)
	if err := pipe.RunTimeout(p, defaultCmdExecTimeout); err != nil {
		return fmt.Errorf("unable to kill tmux session: %w", err)
	}
	return nil
}

// ListSessions returns the session identifiers of the running sessions started by
// pmux. Valid partial results may be returned (i.e. even though the error returned
// is not nil, the list of session identifiers up to that point may be valid).
func ListSessions() ([]string, error) {
	p := pipe.Exec("tmux", "list-sessions")
	acc := []string{}

	stdout, stderr, err := pipe.DividedOutputTimeout(p, defaultCmdExecTimeout)
	if err != nil {
		return acc, fmt.Errorf("unable to list tmux sessions: %w, %v", err, string(stderr))
	}
	if len(stdout) == 0 {
		return acc, nil
	}
	buf := bytes.NewBuffer(stdout)
	s := bufio.NewScanner(buf)
	for s.Scan() {
		line := s.Text()
		sid := strings.Split(line, ":")[0]
		if err = validateSID(sid); err != nil {
			log.Printf("[WARN] ListSessions: skipping line <%v>: %v", line, err)
			continue
		}
		acc = append(acc, string(sid))
	}
	if s.Err() != nil {
		return acc, fmt.Errorf("something went wrong while parsing list-sessions output: %w", err)
	}

	return acc, nil
}

// HasSession returns true if tmux is running a session named "sid".
func HasSession(sid string) bool {
	sessions, err := ListSessions()
	if err != nil {
		log.Printf("[ERROR] HasSession: %v", err)
		return false
	}

	for _, v := range sessions {
		if v == sid {
			return true
		}
	}
	return false
}
