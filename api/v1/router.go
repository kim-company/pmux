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

package v1

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kim-company/pmux/api"
)

// NewRouter creates a `mux.Router` instance adding its routes
// under "api/v1".
func NewRouter() *mux.Router {
	return api.NewRouter(&router{})
}

// router is an ``api.ChildMux'' implementation.
type router struct {
}

func (r *router) PathPrefix() string {
	return "/api/v1"
}

func (r *router) AttachRoutes(root *mux.Router) {
	h := &sessionHandler{}
	root.Path("/sessions").Handler(h).Methods("GET", "POST")
	root.Path("/sessions/{sid}").Handler(h).Methods("GET", "DELETE")
}

type sessionHandler struct {
}

func (t *sessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		t.Create(w, r)
	case "DELETE":
		t.Delete(w, r)
	case "GET":
		sid := mux.Vars(r)["sid"]
		if sid != "" {
			t.Show(w, r, sid)
		} else {
			t.List(w, r)
		}
	default:
		serveError(w, fmt.Errorf("unsupported method %v", r.Method), http.StatusBadRequest)
	}
}

func serveError(w http.ResponseWriter, err error, code int) {
	log.Printf("[ERROR] [code %d] %v", code, err)
	http.Error(w, err.Error(), code)
}

func (t *sessionHandler) List(w http.ResponseWriter, r *http.Request) {
}

func (t *sessionHandler) Create(w http.ResponseWriter, r *http.Request) {
}

func (t *sessionHandler) Show(w http.ResponseWriter, r *http.Request, sid string) {
}

func (t *sessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
}
