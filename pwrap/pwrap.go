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

// Package pwrap provides a process wrapper that is suitable to be executed
// inside a tmux session, allowing to later retriving its stdout, stderr and
// initial configuration.
package pwrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kim-company/pmux/tmux"
)

// PWrap is a process wrapper.
type PWrap struct {
	sid     string
	name    string
	rootDir string
}

// Opt defines the signature of the a configuration function used
// when creating new instances of ``PWrap'' with ``New''.
type Opt func(*PWrap) error

const (
	FileStderr = "stderr"
	FileStdout = "stdout"
	FileConfig = "config"
	FileSID    = "sid"
)

func RootDir(path string) Opt {
	return func(p *PWrap) error {
		// MkdirAll will not do anything if the directory is already there.
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
		files := []string{FileStderr, FileStdout, FileConfig, FileSID}
		for _, v := range files {
			path := filepath.Join(path, v)
			if _, err := os.Stat(path); err == nil {
				// In this case we want to stop: file already exists.
				continue
			}

			f, err := os.Create(path)
			if err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
		p.rootDir = path
		return nil
	}
}

// New is used to instantiate new PWrap instances.
func New(ename string, opts ...Opt) (*PWrap, error) {
	// Is "ename" visible?
	if _, err := exec.LookPath(ename); err != nil {
		return nil, fmt.Errorf("process wrapper validation error: %w", err)
	}

	pw := &PWrap{name: ename}
	for _, f := range opts {
		if err := f(pw); err != nil {
			return nil, fmt.Errorf("unable to apply option on process wrapper initialization: %w", err)
		}
	}
	// After options have been applied, validate the process
	// wrapper before returning it.
	if pw.rootDir == "" {
		return nil, fmt.Errorf("process wrapper validation error: root directory was not set")
	}

	return pw, nil
}

// Path returns the full path of the file as if it were inside "p"'s root
// directory. Returns an error if the file is not present in the directory.
func (p *PWrap) Path(rel string) (string, error) {
	path := filepath.Join(p.rootDir, rel)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

// Open opens a file that must be present in "p"'s root directory. Returns an
// error otherwise. It is caller's responsibility to close the file.
func (p *PWrap) Open(rel string) (*os.File, error) {
	path, err := p.Path(rel)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
}

// StartSession starts the process wrapper in a tmux session. There is not guarantee that the process
// will still be running after this function returns. The session identifier returned will be
// stored indide the relative ``FileSID'' file. This function is a non blocking function.
func (p *PWrap) StartSession() (string, error) {
	sid := tmux.NewSID()
	f, err := p.Open(FileSID)
	if err != nil {
		return "", fmt.Errorf("could not start process wrapper session: %w", err)
	}
	defer f.Close()
	_, err = f.Write([]byte(sid))
	if err != nil {
		return "", fmt.Errorf("could not write session identifier: %w", err)
	}
	if err = tmux.NewSession(sid, "pmux", "wrap", p.name, "--root="+p.rootDir); err != nil {
		return "", fmt.Errorf("could not start process wrapper session: %w", err)
	}

	p.sid = sid
	return sid, nil
}

// Run executes "p"'s command and waits for it to exit. Its stderr and stdout pipes are
// connected to their relative files inside process's root directory.
// The underlying program is executed running `<ename> --config=<configuration file path>`.
func (p *PWrap) Run() error {
	stderr, err := p.Open(FileStderr)
	if err != nil {
		return err
	}
	defer stderr.Close()
	stdout, err := p.Open(FileStdout)
	if err != nil {
		return err
	}
	defer stdout.Close()
	configPath, err := p.Path(FileConfig)
	if err != nil {
		return err
	}

	cmd := exec.Command(p.name, "--config="+configPath)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	return cmd.Run()
}

// Trash removes any traces of the process from the system.
func (p *PWrap) Trash() error {
	if p.sid != "" {
		if err := tmux.KillSession(p.sid); err != nil {
			return fmt.Errorf("unable to trash process wrapper: %w", err)
		}
	}
	return p.trashFiles()
}

func (p *PWrap) trashFiles() error {
	expected := []string{FileStderr, FileStdout, FileConfig, FileSID}
	found := 0
	filepath.Walk(p.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		found++
		for _, v := range expected {
			if filepath.Base(path) == v {
				return os.RemoveAll(path)
			}
		}
		return nil

	})
	if found == len(expected)+1 /* 1 for the directory itself */ {
		return os.RemoveAll(p.rootDir)
	}

	return nil
}
