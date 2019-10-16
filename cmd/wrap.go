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
	"log"

	"github.com/kim-company/pmux/http/pwrapapi"
	"github.com/kim-company/pmux/pwrap"
	"github.com/kim-company/pmux/tmux"
	"github.com/spf13/cobra"
)

var rootDir, sid string
var modeRaw int

// wrapCmd represents the pwrap command
var wrapCmd = &cobra.Command{
	Use:   "wrap",
	Short: "Execute programs inside a wrapper suitable for interacting with pmux",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: errors are difficult to be detected if this process
		// started from a `pwrap.Start()` call, as it means that we're running
		// in a sandboxed tmux session.

		name := args[0]
		mode := pwrapapi.ServerMode(modeRaw)
		pw, err := pwrap.New(pwrap.ExecName(name), pwrap.Mode(mode), pwrap.OverrideSID(sid), pwrap.RootDir(rootDir))
		if err != nil {
			log.Fatal(err)
		}
		if err := pw.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(wrapCmd)
	wrapCmd.Flags().StringVarP(&rootDir, "root", "r", "", "Root process sandbox directory.")
	wrapCmd.Flags().StringVarP(&sid, "sid", "", tmux.NewSID(), "Override session identifier.")
	wrapCmd.Flags().IntVarP(&modeRaw, "mode", "m", int(pwrapapi.ModeNormal), "Set mode type.")
}
