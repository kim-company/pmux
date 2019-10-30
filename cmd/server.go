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
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/kim-company/pmux/http/pmuxapi"
	"github.com/spf13/cobra"
)

var port int
var execName string
var childArgsRaw string
var dirty bool

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		r := pmuxapi.NewRouter(execName,
			pmuxapi.Args(strings.Split(childArgsRaw, ",")),
			pmuxapi.KeepFiles(dirty),
		)
		srv := &http.Server{
			Addr:         fmt.Sprintf("0.0.0.0:%d", port),
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      r,
		}
		// Run our server in a goroutine so that it doesn't block.
		log.Printf("Port: %d, Executable: %s", port, execName)
		log.Printf("Server listening...")
		go func() {
			if err := srv.ListenAndServe(); err != nil {
				log.Println(err)
			}
		}()

		c := make(chan os.Signal, 1)

		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
		// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		signal.Notify(c, os.Interrupt)

		// Block until we receive our signal.
		<-c

		// Create a deadline to wait for.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.
		srv.Shutdown(ctx)
		// Optionally, you could run srv.Shutdown in a goroutine and block on
		// <-ctx.Done() if your application should wait for other services
		// to finalize based on context cancellation.
		log.Println("Shutting down...")
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func init() {
	serverCmd.Flags().IntVarP(&port, "port", "p", 4002, "Server listening port.")
	serverCmd.Flags().StringVarP(&execName, "exec-name", "n", "bin/mockcmd", "Pmux will spawn sessions running this executable.")
	serverCmd.Flags().StringVarP(&childArgsRaw, "args", "", "", "Comma separated list of arguments that pmux will use togheter with \"execName\".")
	serverCmd.Flags().BoolVarP(&dirty, "dirty", "", false, "Enables dirty mode: all files created by pmux child processes are kept.")
}
