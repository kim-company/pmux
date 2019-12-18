// SPDX-FileCopyrightText: 2019 KIM KeepInMind GmbH
//
// SPDX-License-Identifier: MIT

package pwrapapi

import (
	"fmt"
	"net/http"
)

// Server is an http.Server implementation which allows to interact with a local
// process through HTTP.
// Each server tracks only one child cmd.
type Server struct {
	*http.Server
	port int
	r    *Router
}

func CmdSockPath(path string) func(*Server) {
	return func(s *Server) {
		RouteProgress(path)(s.r)
		// TODO: Add also command route to deliver commands.
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
	r := NewRouter()
	s := &Server{r: r}
	for _, f := range opts {
		f(s)
	}

	s.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.r,
	}
	return s
}
