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

package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kim-company/pmux/pwrap"
	"github.com/kim-company/pmux/tmux"
	"github.com/spf13/cobra"
)

var rootDir, sid, url string

// wrapCmd represents the pwrap command
var wrapCmd = &cobra.Command{
	Use:   "wrap",
	Short: "Execute programs inside a wrapper suitable for interacting with pmux",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: errors are difficult to be detected if this process
		// started from a `pwrap.StartSession()` call, as it means that we're running
		// in a sandboxed tmux session.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Note: tmux sends SIGHUP to all child processes when the session
		// is terminated. Children need to be killed when that happens.
		srx := make(chan os.Signal, 1)
		signal.Notify(srx, syscall.SIGHUP, os.Interrupt)
		go func() {
			s := <-srx
			log.Printf("[INFO] signal %v received. Exiting...", s)
			cancel()
		}()

		pw, err := pwrap.New(
			pwrap.Exec(args[0], args[1:]...),
			pwrap.OverrideSID(sid),
			pwrap.RootDir(rootDir),
			pwrap.Register(url),
		)
		if err != nil {
			log.Fatal(err)
		}
		if err := pw.Run(ctx); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(wrapCmd)
	wrapCmd.Flags().StringVarP(&rootDir, "root", "", "", "Root process sandbox directory.")
	wrapCmd.Flags().StringVarP(&sid, "sid", "", tmux.NewSID(), "Override session identifier.")
	wrapCmd.Flags().StringVarP(&url, "reg-url", "", "", "Set registration URL to contact before running the task.")
}
