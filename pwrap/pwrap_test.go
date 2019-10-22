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
	pathSock := filepath.Join(path, FileSock)
	paths := []string{pathStderr, pathStdout, pathConfig, pathSID, pathSock}
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

	_, err := New(ExecName("nxtfxxnd"))
	if (err != nil && !errors.Is(err, exec.ErrNotFound)) || err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = New(ExecName("yes"))
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
	pw, err := New(RootDir(path))
	if err != nil {
		t.Fatal(err)
	}

	path, err = pw.Path(FileStderr)
	if err != nil {
		t.Fatal(err)
	}
	path, err = pw.Path("invalid-file")
	if err == nil {
		t.Fatalf("Expected path error, found nil. Path returned: %v", path)
	}
}
