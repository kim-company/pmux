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
	"net/http"
	"io"

	"github.com/gorilla/mux"
)

// ExecType defines which type of commands the executed child program will be able
// to accept.
type ExecType int

const (
	// Standard set of commands, i.e. no start or stop, the program will just
	// execute its task and exit on its own.
	ExecTypeNormal ExecType = iota
	// The program will spawn and do nothing, waiting for another process to start
	// terminate it.
	ExecTypeLive
)

type Server struct {
	*http.Server
	execType ExecType
	stderr, stdout io.Reader
	sockPath string
}

func Type(t ExecType) func(*Server) {
	return func(s *Server) {
		s.execType = t
	}
}

func Stderr(r io.Reader) func(*Server) {
	return func(s *Server) {
		s.stderr = r
	}
}

func Stdout(r io.Reader) func(*Server) {
	return func(s *Server) {
		s.stdout = r
	}
}

func SockPath(p string) func(*Server) {
	return func(s *Server) {
		s.sockPath = p
	}
}

func NewServer(opts ...func(*Server)) *Server {
	r := mux.NewRouter()
	s := &Server{
		Server: &http.Server{
			Addr:         ":0",
			Handler:      r,
		},
	}
	for _, f := range opts {
		f(s)
	}
	return s
}
