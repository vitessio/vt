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
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/vitessio/vt/go/web"
)

//nolint:gochecknoglobals // FIXME
var wg sync.WaitGroup

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// rootCmd represents the base command when called without any subcommands
	var port int64
	webserverStarted := false
	root := &cobra.Command{
		Use:   "vt",
		Short: "Utils tools for testing, running and benchmarking Vitess.",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Do something with port here
			if port > 0 {
				wg.Add(1)
				if webserverStarted {
					return nil
				}
				webserverStarted = true
				go startWebServer(port)
				time.Sleep(1 * time.Hour)
			}
			return nil
		},
	}
	root.PersistentFlags().Int64VarP(&port, "port", "p", 8080, "Port to run the web server on")

	root.CompletionOptions.HiddenDefaultCmd = true

	root.AddCommand(summarizeCmd())
	root.AddCommand(testerCmd())
	root.AddCommand(tracerCmd())
	root.AddCommand(keysCmd())
	root.AddCommand(dbinfoCmd())
	root.AddCommand(transactionsCmd())
	root.AddCommand(planalyzeCmd())

	if !webserverStarted && port > 0 {
		wg.Add(1)
		webserverStarted = true
		go startWebServer(port)
	}
	err := root.Execute()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	wg.Wait()
}

func launchWebServer(ch chan int, port int64) {
	go func() {
		web.Run(port)
		ch <- 1
	}()
}

func startWebServer(port int64) {
	defer wg.Done()
	if port > 0 && port != 8080 {
		panic("(FIXME: make port configurable) Port is not 8080")
	}
	ch := make(chan int, 2)
	launchWebServer(ch, port)
	if os.WriteFile("/dev/stderr", []byte("Web server is running, use Ctrl-C to exit\n"), 0o600) != nil {
		panic("Failed to write to /dev/stderr")
	}
	<-ch
}
