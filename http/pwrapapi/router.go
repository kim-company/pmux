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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
}

func RouteProgress(path string) func(*Router) {
	return func(r *Router) {
		r.HandleFunc("/progress", progressStreamHandler(path)).Methods("GET")
		r.HandleFunc("/command", commandHandler(path)).Methods("POST")
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

func serveError(w http.ResponseWriter, err error, status int) {
	logError(err, status)
	http.Error(w, err.Error(), status)
}

func logError(err error, status int) {
	log.Printf("[ERROR] [STATUS %d] %v", status, err)
}

func progressStreamHandler(sockPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sock, err := net.Dial("unix", sockPath)
		if err != nil {
			serveError(w, fmt.Errorf("unable to open progress socket: %w", err), http.StatusInternalServerError)
			return
		}
		header := []byte("mode=progress\n")
		sock.Write(header)
		defer sock.Close()
		hijackCopy(w, sock, "text/csv")
	}
}

func commandHandler(sockPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		sock, err := net.Dial("unix", sockPath)
		if err != nil {
			io.Copy(ioutil.Discard, r.Body)
			serveError(w, fmt.Errorf("unable to open progress socket: %w", err), http.StatusInternalServerError)
			return
		}
		defer sock.Close()

		w.WriteHeader(http.StatusOK)
		buf := bytes.NewBuffer([]byte("mode=command\n"))
		_, err = io.Copy(buf, r.Body)
		if err != nil {
			logError(fmt.Errorf("unable to complete copy: %w", err), http.StatusInternalServerError)
			return
		}
		buf.Write([]byte("\n"))
		_, err = io.Copy(sock, buf)
		if err != nil {
			logError(fmt.Errorf("unable to complete copy: %w", err), http.StatusInternalServerError)
			return
		}
	}
}

func hijackCopy(w http.ResponseWriter, src io.Reader, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	// Hijack the connection for uninterrupted data stream delivery.
	hj, ok := w.(http.Hijacker)
	if !ok {
		serveError(w, fmt.Errorf("webserver doesn't support hijacking"), http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		serveError(w, fmt.Errorf("unable to hijack connection: %w", err), http.StatusInternalServerError)
		return
	}
	cw := httputil.NewChunkedWriter(conn)
	defer conn.Close()
	defer cw.Close()

	n, err := io.Copy(cw, src)
	if err != nil {
		logError(fmt.Errorf("unable to complete copy: %w", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] copy: #%d bytes transferred", n)
}
