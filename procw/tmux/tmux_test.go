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

package tmux

import (
	"strings"
	"testing"
)

func TestHasSession(t *testing.T) {
	sid := NewSID()
	if HasSession(sid) {
		t.Fatalf("session <%s> SHOULD NOT BE present", sid)
	}

	// Start a session using the "yes" utility.
	// From yes manual:
	// yes outputs expletive, or, by default, ``y'', forever.
	if err := NewSession(sid, "yes"); err != nil {
		t.Fatal(err)
	}
	sessions, err := ListSessions()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Sessions: %v", sessions)
	if HasSession(sid) {
		t.Fatalf("session <%s> SHOULD BE present", sid)
	}

	// Now kill this session and repeat the checks.
	if err := KillSession(sid); err != nil {
		t.Fatal(err)
	}

	if HasSession(sid) {
		t.Fatalf("session <%s> SHOULD NOT BE present", sid)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	v, err := Version()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("tmux version: %v", v)
	if !strings.HasPrefix(v, "tmux") {
		t.Fatal("tmux version is expected to start with \"tmux\"")
	}
}

func TestVerify(t *testing.T) {
	t.Parallel()
	if err := Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSID(t *testing.T) {
	var err error
	err = validateSID("pmux-abc")
	if err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
	sid := "invalid-sid"
	err = validateSID(sid)
	if err == nil {
		t.Fatalf("Expected sid validation error for <%v>", sid)
	}
}
