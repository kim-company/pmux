// Copyright 2019 KIM Keep In Mind GmbH //
// SPDX-FileCopyrightText: 2019 KIM KeepInMind GmbH
//
// SPDX-License-Identifier: MIT

package tmux

import (
	"strings"
	"testing"
)

func TestHasSession(t *testing.T) {
	t.Parallel()

	sid := NewSID()
	if HasSession(sid) {
		t.Fatalf("session <%s> SHOULD NOT BE present", sid)
	}

	if err := NewSession(sid, "sleep", "60"); err != nil {
		t.Fatal(err)
	}
	if !HasSession(sid) {
		t.Fatalf("Session <%s> SHOULD BE present", sid)
	}

	// Now kill this session and repeat the checks.
	if err := KillSession(sid); err != nil {
		t.Fatal(err)
	}

	if HasSession(sid) {
		t.Fatalf("Session <%s> SHOULD NOT BE present", sid)
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
	if err := verify(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSID(t *testing.T) {
	var err error
	err = validateSID("pmux-f2dcf053-0966-4d51-984e-0a4de0f0b0d6")
	if err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
	sid := "invalid-sid"
	err = validateSID(sid)
	if err == nil {
		t.Fatalf("Expected sid validation error for <%v>", sid)
	}
}
