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

package pmuxapi

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
	keepFiles bool
	execName  string
	args      []string
}

func KeepFiles(ok bool) func(*Router) {
	return func(r *Router) {
		r.keepFiles = ok
	}
}

func Args(args []string) func(*Router) {
	return func(r *Router) {
		r.args = args
	}
}

// NewRouter returns a new ``Router'' instance which satisfies the ``http.Handler''
// interface.
func NewRouter(execName string, opts ...func(*Router)) *Router {
	r := &Router{Router: mux.NewRouter()}

	r.Use(loggingMiddleware)
	r.HandleFunc("/health_check", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Online!")
	}).Methods("GET")

	// Apply options on router.
	for _, f := range opts {
		f(r)
	}

	h := &SessionHandler{}
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/sessions", h.HandleList()).Methods("GET")
	v1.HandleFunc("/sessions", h.HandleCreate(execName, r.args...)).Methods("POST")
	v1.HandleFunc("/sessions/{sid}", h.HandleDelete(r.keepFiles)).Methods("DELETE")

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
