/*
Copyright 2024 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/vitessio/vt/go/web"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// rootCmd represents the base command when called without any subcommands
	var port int64
	webserverStarted := false
	ch := make(chan int, 2)
	root := &cobra.Command{
		Use:   "vt",
		Short: "Utils tools for testing, running and benchmarking Vitess.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if port > 0 {
				if webserverStarted {
					return nil
				}
				webserverStarted = true
				go startWebServer(port, ch)
				<-ch
			}
			return nil
		},
	}
	root.PersistentFlags().Int64VarP(&port, "port", "p", 8080, "Port to run the web server on")
	root.CompletionOptions.HiddenDefaultCmd = true

	root.AddCommand(summarizeCmd(&port))
	root.AddCommand(testerCmd())
	root.AddCommand(tracerCmd())
	root.AddCommand(keysCmd())
	root.AddCommand(dbinfoCmd())
	root.AddCommand(transactionsCmd())
	root.AddCommand(planalyzeCmd())

	if err := root.ParseFlags(os.Args[1:]); err != nil {
		panic(err)
	}

	if !webserverStarted && port > 0 {
		webserverStarted = true
		go startWebServer(port, ch)
	} else {
		ch <- 1
	}

	// FIXME: add sync b/w webserver and root command, for now just add a wait to make sure webserver is running
	time.Sleep(2 * time.Second)

	err := root.Execute()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	<-ch
}

func startWebServer(port int64, ch chan int) {
	if port > 0 && port != 8080 {
		panic("(FIXME: make port configurable) Port is not 8080")
	}
	web.Run(port)
	if os.WriteFile("/dev/stderr", []byte("Web server is running, use Ctrl-C to exit\n"), 0o600) != nil {
		panic("Failed to write to /dev/stderr")
	}
	ch <- 1
}
