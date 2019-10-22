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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/kim-company/pmux/pwrap"
	"github.com/kim-company/pmux/tmux"
)

type SessionHandler struct {
}

func (h *SessionHandler) writeResponse(w http.ResponseWriter, p interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(p); err != nil {
		h.writeError(w, fmt.Errorf("unable to encode respone: %w", err), http.StatusInternalServerError)
		return err
	}
	return nil
}

func (h *SessionHandler) writeError(w http.ResponseWriter, err error, status int) {
	log.Printf("[ERROR] [STATUS %d] %v", status, err)
	http.Error(w, err.Error(), status)
}

func (h *SessionHandler) HandleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := tmux.ListSessions()
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
		h.writeResponse(w, sessions)
	}
}

var rootDir = filepath.Join(os.TempDir(), "pmux", "sessionsd")

func (h *SessionHandler) HandleCreate(execName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var c struct {
			URL    string      `json:"register_url"`
			Config interface{} `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			h.writeError(w, fmt.Errorf("unable to decode create payload body: %w", err), http.StatusInternalServerError)
			return
		}

		pw, err := pwrap.New(pwrap.ExecName(execName), pwrap.RootDir(rootDir), pwrap.Register(c.URL))
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
		configFile, err := pw.Open(pwrap.FileConfig, os.O_RDWR|os.O_CREATE)
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			pw.Trash()
			return
		}
		defer configFile.Close()
		if err := json.NewEncoder(configFile).Encode(c.Config); err != nil {
			h.writeError(w, fmt.Errorf("unable to store configuration: %w", err), http.StatusInternalServerError)
			pw.Trash()
			return
		}

		log.Printf("[INFO] Starting [%v] session, working dir: %v", execName, pw.WorkDir())
		sid, err := pw.StartSession()
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			pw.Trash()
			return
		}
		if err = h.writeResponse(w, sid); err != nil {
			pw.Trash()
		}
	}
}

func (h *SessionHandler) HandleDelete(keepFiles bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := mux.Vars(r)["sid"]
		if sid == "" {
			h.writeError(w, fmt.Errorf("unable to retrieve session identifier from request context"), http.StatusBadRequest)
			return
		}

		pw, err := pwrap.New(pwrap.OverrideSID(sid), pwrap.RootDir(rootDir))
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}

		deleteFunc := pw.Trash
		if keepFiles {
			deleteFunc = pw.KillSession
		}
		if err = deleteFunc(); err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
		h.writeResponse(w, sid)
	}
}
