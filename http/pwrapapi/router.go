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
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
}

func RouteStderr(path string) func(*Router) {
	return func(r *Router) {
		r.HandleFunc("/stderr", stderrStreamHandler(path)).Methods("GET")
	}
}

func RouteStdout(path string) func(*Router) {
	return func(r *Router) {
		r.HandleFunc("/stdout", stdoutStreamHandler(path)).Methods("GET")
	}
}

func RouteProgress(path string) func(*Router) {
	return func(r *Router) {
		r.HandleFunc("/progress", progressStreamHandler(path)).Methods("GET")
	}
}

func NewRouter(opts ...func(*Router)) *Router {
	r := &Router{Router: mux.NewRouter()}
	r.Use(loggingMiddleware)
	r.HandleFunc("/health_check", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Online!")
	}).Methods("GET")

	for _, f := range opts {
		f(r)
	}
	return r
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Printf("[%v] %v", r.Method, r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, err error, status int) {
	log.Printf("[ERROR] [STATUS %d] %v", status, err)
	http.Error(w, err.Error(), status)
}

func stderrStreamHandler(stderrPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(stderrPath)
		if err != nil {
			writeError(w, fmt.Errorf("unable to open stderr: %w", err), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		hijackCopy(w, f, "text/plain")
	}
}

func stdoutStreamHandler(stdoutPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(stdoutPath)
		if err != nil {
			writeError(w, fmt.Errorf("unable to open stdout: %w", err), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		hijackCopy(w, f, "text/plain")
	}
}

func progressStreamHandler(sockPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sock, err := net.Dial("unix", sockPath)
		if err != nil {
			writeError(w, fmt.Errorf("unable to open progress socket: %w", err), http.StatusInternalServerError)
			return
		}
		defer sock.Close()
		hijackCopy(w, sock, "text/csv")
	}
}

func hijackCopy(w http.ResponseWriter, src io.Reader, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	// Hijack the connection for uninterrupted data stream delivery.
	hj, ok := w.(http.Hijacker)
	if !ok {
		writeError(w, fmt.Errorf("webserver doesn't support hijacking"), http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		writeError(w, fmt.Errorf("unable to hijack connection: %w", err), http.StatusInternalServerError)
		return
	}
	cw := httputil.NewChunkedWriter(conn)
	defer conn.Close()
	defer cw.Close()

	io.Copy(cw, src)
}
