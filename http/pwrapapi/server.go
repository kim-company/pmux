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
)

// Server is an http.Server implementation which allows to interact with a local
// process through HTTP.
// Each server instance holds exactly one child process.
type Server struct {
	*http.Server
	port  int
	child *cmdSettings
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
	s := &Server{child: &cmdSettings{}}
	for _, f := range opts {
		f(s)
	}
	r := newRouter(s.child)
	s.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: r,
	}
	return s
}
