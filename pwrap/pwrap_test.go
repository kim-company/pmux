// SPDX-FileCopyrightText: 2019 KIM KeepInMind GmbH
//
// SPDX-License-Identifier: MIT

package pwrap

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	t.Parallel()

	pw, err := New(OverrideSID(uuid.New().String()), RootDir(os.TempDir()))
	if err != nil {
		t.Fatal(err)
	}

	path := pw.WorkDir()
	pathStderr := filepath.Join(path, FileStderr)
	pathStdout := filepath.Join(path, FileStdout)
	pathConfig := filepath.Join(path, FileConfig)
	pathSID := filepath.Join(path, FileSID)
	paths := []string{pathStderr, pathStdout, pathConfig, pathSID}
	for _, v := range paths {
		if _, err := os.Stat(v); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

func TestNew_ExecName(t *testing.T) {
	t.Parallel()

	_, err := New(Exec("nxtfxxnd"))
	if (err != nil && !errors.Is(err, exec.ErrNotFound)) || err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = New(Exec("yes"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestTrashFiles(t *testing.T) {
	t.Parallel()

	pw, err := New(RootDir(os.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	path := pw.WorkDir()

	// In this case, trash files should destroy the whole
	// directory.
	if err := pw.trashFiles(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); (err != nil && !errors.Is(err, os.ErrNotExist)) || err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// If instead there is something else in the directory, the directory
	// and that something else should still be there.
	path = filepath.Join(os.TempDir(), "pwrap-test-"+uuid.New().String())
	pw, err = New(RootDir(path))
	if err != nil {
		t.Fatal(err)
	}
	extraPath := filepath.Join(path, "extra-file")
	f, err := os.Create(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	if err := pw.trashFiles(); err != nil {
		t.Fatal(err)
	}
	// Directory itself should be there.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Also the extra path file.
	if _, err := os.Stat(extraPath); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Other pwrap files should be disappeared.
	if _, err := os.Stat(filepath.Join(path, FileConfig)); (err != nil && !errors.Is(err, os.ErrNotExist)) || err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(os.TempDir(), "pwrap-test")
	sid := "1234"
	pw, err := New(OverrideSID(sid), RootDir(path))
	if err != nil {
		t.Fatal(err)
	}

	stderrPath := pw.Path(FileStderr)
	expStderrPath := filepath.Join(path, sid, FileStderr)
	if stderrPath != expStderrPath {
		t.Fatalf("Wanted %v, found %v", expStderrPath, stderrPath)
	}
}
