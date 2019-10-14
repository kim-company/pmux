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
}

func NewRouter() *Router {
	r := &Router{mux.NewRouter()}

	r.Use(loggingMiddleware)
	r.HandleFunc("/health_check", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Online!")

	}).Methods("GET")

	// API v1
	// TODO: flexibility!
	h := &SessionHandler{}
	v1 := r.PathPrefix("api/v1")
	v1.HandlerFunc(h.HandleList()).Methods("GET").Path("/sessions")
	v1.HandlerFunc(h.HandleCreate("yes")).Methods("POST").Path("/sessions")
	v1.HandlerFunc(h.HandleDelete(false)).Methods("DELETE").Path("/sessions/{sid}")

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
