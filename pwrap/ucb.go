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

package pwrap

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// UnixCommBridge is a Unix Socket listener that extends the communication channels
// available by wrapped commands. If a command is capable of providing updates
// about is progress in completing the task, it can use this socket to write
// updates to it and receive commands from it.
// The socket can be used to enable the communication between the child process
// and the process wrapper, which will expose the socket to the internet through
// its HTTP API.
type UnixCommBridge struct {
	path string
	net.Listener
	last struct {
		sync.Mutex
		u *string
	}
	clients struct {
		sync.Mutex
		m map[string]chan string
	}

	onCommand func(*UnixCommBridge, string) error
}

// OnCommand sets the onCommand function option. When a command is recevied through the socket,
// this handler will be called.
func OnCommand(h func(*UnixCommBridge, string) error) func(*UnixCommBridge) {
	return func(u *UnixCommBridge) {
		u.onCommand = h
	}
}

// NewUnixCommBridge starts a Unix Domain Socket listener on ``path''.
// Is is the caller's responsibility to close the listener when it's done.
func NewUnixCommBridge(ctx context.Context, path string, opts ...func(*UnixCommBridge)) (*UnixCommBridge, error) {
	l, err := new(net.ListenConfig).Listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("unable to listen on %v: %w", path, err)
	}
	u := &UnixCommBridge{Listener: l, path: path}
	for _, f := range opts {
		f(u)
	}
	return u, nil
}

func (b *UnixCommBridge) Open(ctx context.Context) {
	for {
		conn, err := b.Listener.Accept()
		if err != nil {
			log.Printf("[ERROR] unable to accept more connections: %v", err)
			return
		}

		go b.handleConn(ctx, conn)
	}
}

func (b *UnixCommBridge) Close() error {
	defer os.Remove(b.path)
	return b.Listener.Close()
}

func (b *UnixCommBridge) Write(p []byte) (int, error) {
	s := string(p)

	b.last.Lock()
	b.last.u = &s
	b.last.Unlock()

	b.clients.Lock()
	defer b.clients.Unlock()
	for _, v := range b.clients.m {
		v <- s
	}
	return len(p) * len(b.clients.m), nil

}

type tx struct {
	close func()
	c     <-chan string
}

func (b *UnixCommBridge) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	header, err := r.ReadString('\n')
	if err != nil {
		log.Printf("[ERROR] handle unix conn: unable to read header: %v", err)
		return
	}
	log.Printf("[DEBUG] header read: %v", header)
	switch {
	case strings.Contains(header, "mode=command"):
		if err := b.readCommand(ctx, r); err != nil {
			log.Printf("[ERROR] unable to read command: %v", err)
		}
	case strings.Contains(header, "mode=progress"):
		if err := b.writeUpdates(ctx, conn); err != nil {
			log.Printf("[ERROR] unable to write update to connection %v: %v", conn.RemoteAddr().String(), err)
		}
	default:
		log.Printf("[ERROR] handle unix conn: unrecognised header \"%s\"", header)
		return
	}
}

func (b *UnixCommBridge) getTx() *tx {
	c := make(chan string, 1)

	b.last.Lock()
	// generate a timestamp key inside the lock, so we're ensured to receive a unique one.
	key := fmt.Sprintf("%d", time.Now().UnixNano())
	if b.last.u != nil {
		c <- *b.last.u
	}
	b.last.Unlock()

	b.clients.Lock()
	if b.clients.m == nil {
		b.clients.m = make(map[string]chan string)
	}
	b.clients.m[key] = c
	b.clients.Unlock()

	return &tx{
		c: c,
		close: func() {
			close(c)
			b.clients.Lock()
			delete(b.clients.m, key)
			b.clients.Unlock()
		},
	}
}

func (b *UnixCommBridge) writeUpdates(ctx context.Context, w io.Writer) error {
	c := b.getTx()

	defer c.close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case u := <-c.c:
			// Note: If the connection is closed, we will not be able to detect it
			// util the next time that we try to write something into it.
			if _, err := w.Write([]byte(u)); err != nil {
				return err
			}
		}
	}
}

func (b *UnixCommBridge) readCommand(ctx context.Context, r *bufio.Reader) error {
	if b.onCommand == nil {
		return fmt.Errorf("no command handler has been configured")
	}

	cmd, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("unable to read command: %w", err)
	}

	log.Printf("[INFO] command read: %v", cmd)
	return b.onCommand(b, cmd)
}
