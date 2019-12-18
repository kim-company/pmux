// SPDX-FileCopyrightText: 2019 KIM KeepInMind GmbH
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/kim-company/pmux/tmux"
)

func main() {
	sessions, err := tmux.ListSessions()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Printf("Pmux sessions:\n%v\n", sessions)
}
