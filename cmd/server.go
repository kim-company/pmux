// SPDX-FileCopyrightText: 2019 KIM KeepInMind GmbH
//
// SPDX-License-Identifier: MIT

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
		log.Println("Server is shutting down...")
		srv.Shutdown(ctx)
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVarP(&port, "port", "p", 4002, "Server listening port.")
	serverCmd.Flags().StringVarP(&execName, "exec-name", "n", "bin/mockcmd", "Pmux will spawn sessions running this executable.")
	serverCmd.Flags().StringVarP(&childArgsRaw, "args", "", "", "Comma separated list of arguments that pmux will use togheter with \"execName\".")
	serverCmd.Flags().BoolVarP(&dirty, "dirty", "", false, "Enables dirty mode: all files created by pmux child processes are kept.")
}
