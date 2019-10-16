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

package pwrapapi

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// ServerMode defines which type of commands the executed child program will be able
// to accept.
type ServerMode int

const (
	// Standard set of commands, i.e. no start or stop, the child cmd will just
	// execute its task and exit on its own.
	ModeNormal ServerMode = iota
	// The program will spawn and do nothing, waiting for another process to start
	// terminate it.
	ModeLive
)

type cmdSettings struct {
	stderr, stdout io.Reader
	sockPath       string
}

// Server is an http.Server implementation which allows to interact with a local
// process through HTTP.
// Each server instance holds exactly one child process.
type Server struct {
	*http.Server
	port  int
	mode  ServerMode
	child cmdSettings
}

// Mode sets the mode option.
func Mode(m ServerMode) func(*Server) {
	return func(s *Server) {
		s.mode = m
	}
}

// ChildStderr sets the child stderr option.
func ChildStderr(r io.Reader) func(*Server) {
	return func(s *Server) {
		s.child.stderr = r
	}
}

// ChildStdout sets the child stdout option.
func ChildStdout(r io.Reader) func(*Server) {
	return func(s *Server) {
		s.child.stdout = r
	}
}

// ChildSockPath sets the child sock path option.
func ChildSockPath(p string) func(*Server) {
	return func(s *Server) {
		s.child.sockPath = p
	}
}

// Port sets server's listening port option.
func Port(p int) func(*Server) {
	return func(s *Server) {
		s.port = p
	}
}

// NewServer creates a new Server instance.
func NewServer(opts ...func(*Server)) *Server {
	r := mux.NewRouter()
	s := &Server{child: cmdSettings{}}
	for _, f := range opts {
		f(s)
	}
	s.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: r,
	}
	return s
}

func (s *Server) ListenAndServe() error {
	// TODO: register with remote master
	return s.Server.ListenAndServe()
}
