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

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var (
	configPath string
	sockPath   string
)

// mockCmd represents the mockcmd command
var mockCmd = &cobra.Command{
	Use:   "mockcmd",
	Short: "A default mocked command which can be executed by pmux, but does not do anything useful.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stderr, "mockcmd stderr\n")
		fmt.Fprintf(os.Stdout, "mockcmd stdout\n")

		var w io.WriteCloser = os.Stdout
		if sockPath != "" {
			ctx := context.Background()
			l, err := listen(ctx, sockPath)
			if err != nil {
				log.Printf("[ERROR] %v", err)
				return
			}
			w = NewUpdatesWriter(ctx, l)
		}

		for i := 0; ; i++ {
			fmt.Fprintf(w, "waiting %d...", i)
			<-time.After(time.Millisecond * 1000)
			fmt.Fprintf(w, "done!\n")
		}
	},
}

func init() {
	mockCmd.Flags().StringVarP(&configPath, "config", "", "config.json", "Path to the configuration file.")
	mockCmd.Flags().StringVarP(&sockPath, "socket-path", "", "io.sock", "Path to the communication socket address.")
}

const defaultBufferCap = 4096

// Listen starts a Unix Domain Socket listener on ``sockPath''.
// Is is the caller's responsibility to close the listener when it's done.
func listen(ctx context.Context, sockPath string) (net.Listener, error) {
	l, err := new(net.ListenConfig).Listen(ctx, "unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("unable to listen on %v: %w", sockPath, err)
	}
	return l, nil
}

func NewUpdatesWriter(ctx context.Context, l net.Listener) *UpdatesWriter {
	uw := &UpdatesWriter{}
	go uw.accept(ctx, l)
	return uw
}

type UpdatesWriter struct {
	last struct {
		sync.Mutex
		u *string
	}
	clients struct {
		sync.Mutex
		m map[string]chan string
	}
}

func (w *UpdatesWriter) Write(p []byte) (int, error) {
	if err := w.WriteUpdate(string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *UpdatesWriter) Close() error {
	return nil
}

func (w *UpdatesWriter) WriteUpdate(u string) error {
	w.last.Lock()
	w.last.u = &u
	w.last.Unlock()

	w.clients.Lock()
	defer w.clients.Unlock()
	for _, v := range w.clients.m {
		v <- u
	}
	return nil
}

type rx struct {
	close func()
	c     <-chan string
}

func (w *UpdatesWriter) getTx() *rx {
	tx := make(chan string, 1)

	w.last.Lock()
	// generate a timestamp key inside the lock, so we're ensured to receive a unique one.
	key := fmt.Sprintf("%d", time.Now().UnixNano())
	if w.last.u != nil {
		tx <- *w.last.u
	}
	w.last.Unlock()

	w.clients.Lock()
	if w.clients.m == nil {
		w.clients.m = make(map[string]chan string)
	}
	w.clients.m[key] = tx
	w.clients.Unlock()

	return &rx{
		c: tx,
		close: func() {
			close(tx)
			w.clients.Lock()
			delete(w.clients.m, key)
			w.clients.Unlock()
		},
	}
}

func (w *UpdatesWriter) accept(ctx context.Context, l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("[ERROR] uds listener: %v", err)
			return
		}

		rx := w.getTx()
		go serveUpdates(ctx, conn, rx)
	}
}

func serveUpdates(ctx context.Context, conn net.Conn, rx *rx) {
	defer conn.Close()
	defer rx.close()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] closing connection to %v: %v", conn.RemoteAddr().String(), ctx.Err())
		case u := <-rx.c:
			// Note: If the connection is closed, we will not be able to detect it
			// util the next time that we try to write something into it.
			if _, err := conn.Write([]byte(u)); err != nil {
				log.Printf("[ERROR] unable to write update to connection %v: %v", conn.RemoteAddr().String(), err)
				return
			}
		}
	}
}
func main() {
	mockCmd.Execute()
}
