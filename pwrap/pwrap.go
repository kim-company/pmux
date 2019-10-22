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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kim-company/pmux/http/pwrapapi"
	"github.com/kim-company/pmux/tmux"
	"github.com/phayes/freeport"
)

// PWrap is a process wrapper.
type PWrap struct {
	rootDir string
	sid     string
	name    string
	regURL  string
}

// SID returns the assigned session identifier.
func (p *PWrap) SID() string {
	return p.sid
}

// WorkDir returns the current working directory.
func (p *PWrap) WorkDir() string {
	return filepath.Join(p.rootDir, p.sid)
}

// ExecName sets and checks the executable name option.
func ExecName(n string) func(*PWrap) error {
	return func(p *PWrap) error {
		// Is "n" visible?
		if _, err := exec.LookPath(n); err != nil {
			return err
		}
		p.name = n
		return nil
	}
}

// Register sets the register url option.
func Register(url string) func(*PWrap) error {
	return func(p *PWrap) error {
		p.regURL = url
		return nil
	}
}

const (
	FileStderr = "stderr"
	FileStdout = "stdout"
	FileConfig = "config"
	FileSID    = "sid"
	FileSock   = "io.sock"
)

// OverrideSID sets the sid option.
// This function has to be called before "RootDir" if used in the ``New'' function
// in order for it to make effect.
func OverrideSID(sid string) func(*PWrap) error {
	return func(p *PWrap) error {
		p.sid = sid
		return nil
	}
}

// RootDir sets the root directory option.
func RootDir(path string) func(*PWrap) error {
	return func(p *PWrap) error {
		p.rootDir = path
		dir := filepath.Join(path, p.sid)

		// MkdirAll will not do anything if the directory is already there.
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
		files := []string{FileStderr, FileStdout, FileConfig, FileSID, FileSock}
		for _, v := range files {
			file := filepath.Join(dir, v)
			if _, err := os.Stat(file); err == nil {
				// In this case we want to stop: file already exists.
				continue
			}

			f, err := os.Create(file)
			if err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
		return nil
	}
}

// New is used to instantiate new PWrap instances.
func New(opts ...func(*PWrap) error) (*PWrap, error) {
	// Assign executable name and session identifer.
	pw := &PWrap{sid: tmux.NewSID()}

	for _, f := range opts {
		if err := f(pw); err != nil {
			return nil, fmt.Errorf("unable to apply option on process wrapper initialization: %w", err)
		}
	}

	return pw, nil
}

// Path returns the full path of the file as if it were inside "p"'s root
// directory.
func (p *PWrap) Path(rel string) (string, error) {
	path := filepath.Join(p.WorkDir(), rel)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func (p *PWrap) paths(rels ...string) ([]string, error) {
	acc := make([]string, len(rels))
	for i, v := range rels {
		path, err := p.Path(v)
		if err != nil {
			return acc, err
		}
		acc[i] = path
	}
	return acc, nil
}

// Open opens a file that must be present in "p"'s root directory. Returns an
// error otherwise. It is caller's responsibility to close the file.
func (p *PWrap) Open(rel string, flag int) (*os.File, error) {
	path, err := p.Path(rel)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(path, flag, 0755)
}

func (p *PWrap) openMore(flag int, rels ...string) ([]*os.File, error) {
	acc := make([]*os.File, len(rels))
	for i, v := range rels {
		f, err := p.Open(v, flag)
		if err != nil {
			closeAll(acc)
			return []*os.File{}, err
		}
		acc[i] = f
	}
	return acc, nil
}

func closeAll(files []*os.File) {
	for _, v := range files {
		if v != nil {
			v.Close()
		}
	}
}

// StartSession starts the process wrapper in a tmux session. There is not guarantee that the process
// will still be running after this function returns. The session identifier returned will be
// stored indide the relative ``FileSID'' file. This function is a non blocking function.
func (p *PWrap) StartSession() (string, error) {
	sid := p.SID()
	if sid == "" {
		return "", fmt.Errorf("could not start process wrapper session: session identifier not set")
	}

	f, err := p.Open(FileSID, os.O_RDWR|os.O_CREATE)
	if err != nil {
		return "", fmt.Errorf("could not start process wrapper session: %w", err)
	}
	defer f.Close()
	_, err = f.Write([]byte(sid + "\n"))
	if err != nil {
		return "", fmt.Errorf("could not write session identifier: %w", err)
	}
	if err = tmux.NewSession(sid, os.Args[0], "wrap", p.name, "--r="+p.rootDir, "--sid="+sid, "--reg-url="+p.regURL); err != nil {
		return "", fmt.Errorf("could not start process wrapper session: %w", err)
	}

	return sid, nil
}

// KillSession kills the associated tmux session, if any is running.
func (p *PWrap) KillSession() error {
	if p.sid == "" {
		return fmt.Errorf("cannot kill session if process wrapper does not have a session identifier")
	}
	if err := tmux.KillSession(p.sid); err != nil {
		return fmt.Errorf("unable to kill process wrapper session: %w", err)
	}
	p.sid = ""
	return nil
}

// Register performs an HTTP POST request to `regURL`, if present. It registers "port" with the
// remote handler, and returnes a nil error only if the response's status is 200.
func (p *PWrap) Register(port int) error {
	if p.regURL == "" {
		log.Printf("[WARN] registration URL not set")
		return nil
	}

	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(&struct {
		Port int `json:"port"`
	}{
		Port: port,
	}); err != nil {
		return fmt.Errorf("error while building registration payload: %w", err)
	}
	resp, err := http.Post(p.regURL, "application/json", &buf)
	if err != nil {
		return fmt.Errorf("registration error: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed: status code returned is: %d", resp.StatusCode)
	}
	return nil
}

// Run executes "p"'s command and waits for it to exit. Its stderr and stdout pipes are
// connected to their relative files inside process's root directory.
// The underlying program is executed running `<ename> --config=<configuration file path>`.
// If an error occurs, is is both returned and written into wrapper's stderr, if possible.
func (p *PWrap) Run() error {
	if err := p.run(); err != nil {
		f, ferr := p.Open(FileStderr, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
		if ferr != nil {
			return fmt.Errorf("unable to write run error %w, which is: %w", ferr, err)
		}
		if _, werr := f.Write([]byte(err.Error() + "\n")); werr != nil {
			return fmt.Errorf("unable to write run error %w, which is: %w", ferr, err)
		}
		return err
	}
	return nil
}

func (p *PWrap) run() error {
	port, err := freeport.GetFreePort()
	if err != nil {
		return fmt.Errorf("unable to run: failed getting free port: %w", err)
	}
	if err = p.Register(port); err != nil {
		return fmt.Errorf("unable to run: %w", err)
	}

	files, err := p.openMore(os.O_APPEND|os.O_CREATE|os.O_WRONLY, FileStdout, FileStderr)
	if err != nil {
		return fmt.Errorf("unable to run: failed opening stderr and stdout files: %w", err)
	}
	defer closeAll(files)

	paths, err := p.paths(FileConfig, FileSock, FileStdout, FileStderr)
	if err != nil {
		return fmt.Errorf("unable to run: failed retriving necessary paths: %w", err)
	}

	// What we want to accomplish is that if either the API or
	// the tool exit, the other does too.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, p.name, "--config="+paths[0], "--socket-path="+paths[1])
	cmd.Stdout = files[0]
	cmd.Stderr = files[1]

	srv := pwrapapi.NewServer(
		pwrapapi.Port(port),
		pwrapapi.CmdStdoutPath(paths[2]),
		pwrapapi.CmdStderrPath(paths[3]),
		pwrapapi.CmdSockPath(paths[1]),
	)
	errc := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && errors.Is(err, http.ErrServerClosed) {
			// server was closed, i.e. the Run() command exited.
			errc <- nil
			return
		}
		if err != nil {
			// server exited with a critical error
			cancel()
			errc <- err
		}
		errc <- nil
	}()

	err = cmd.Run()
	if err != nil && errors.Is(err, context.Canceled) {
		// It was the server that exited with a critical error
		// apparently.
		if srvErr := <-errc; srvErr != nil {
			return fmt.Errorf("run exited due to a process wrapper API server error: %w", err)
		}
		return fmt.Errorf("run exited with an unexpected error: %w", err)
	}

	// Command exited and the server is still running (teoretically). Shutdown
	// the server before inspecting the error.

	ctx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	<-errc

	if err != nil {
		return fmt.Errorf("run exited with error: %w", err)
	}
	return nil
}

// Trash removes any traces of the process from the system. It even kills the session if any
// is running.
func (p *PWrap) Trash() error {
	if p.sid != "" {
		if err := tmux.KillSession(p.sid); err != nil {
			return fmt.Errorf("unable to trash process wrapper: %w", err)
		}
	}
	return p.trashFiles()
}

func (p *PWrap) trashFiles() error {
	expected := []string{FileStderr, FileStdout, FileConfig, FileSID, FileSock}
	found := 0
	filepath.Walk(p.WorkDir(), func(path string, info os.FileInfo, err error) error {
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
		return os.RemoveAll(p.WorkDir())
	}

	return nil
}
