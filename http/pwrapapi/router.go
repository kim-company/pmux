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
	"net/http"

	"github.com/gorilla/mux"
)

type cmdSettings struct {
	stderr, stdout io.Reader
	sockPath       string
}

type router struct {
	*mux.Router
	*cmdSettings
}

// newRouter returns a new ``router'' instance which satisfies the ``http.Handler''
// interface.
func newRouter(s *cmdSettings) *router {
	r := &router{
		Router:      mux.NewRouter(),
		cmdSettings: s,
	}

	r.Use(loggingMiddleware)
	r.HandleFunc("/health_check", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Online!")
	}).Methods("GET")

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
