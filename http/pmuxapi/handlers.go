// SPDX-FileCopyrightText: 2019 KIM KeepInMind GmbH
//
// SPDX-License-Identifier: MIT

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

func (h *SessionHandler) writeSID(w http.ResponseWriter, sid string) error {
	payload := struct {
		SID string `json:"sid"`
	}{
		SID: sid,
	}
	return h.writeResponse(w, &payload)
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

func (h *SessionHandler) HandleCreate(name string, args ...string) http.HandlerFunc {
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

		pw, err := pwrap.New(pwrap.Exec(name, args...), pwrap.RootDir(rootDir), pwrap.Register(c.URL))
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
		configFile, err := pw.Open(pwrap.FileConfig, os.O_RDWR|os.O_CREATE, os.ModePerm)
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

		log.Printf("[INFO] Starting [%v] session, working dir: %v", name, pw.WorkDir())
		sid, err := pw.StartSession()
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			pw.Trash()
			return
		}
		if err = h.writeSID(w, sid); err != nil {
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
		h.writeSID(w, sid)
	}
}
