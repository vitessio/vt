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
	"github.com/vitessio/vt/go/web/state"
)

//nolint:gochecknoglobals // the state is protected using mutexes
var wstate *state.State

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// rootCmd represents the base command when called without any subcommands
	root := &cobra.Command{
		Use:   "vt",
		Short: "Utils tools for testing, running and benchmarking Vitess.",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	var port int64
	root.PersistentFlags().Int64VarP(&port, "port", "p", 8080, "Port to run the web server on")

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

	// Start the web server for all commands no matter what
	wstate = state.NewState(port)
	ch := make(chan int, 1)
	if port > 0 {
		wstate.SetStarted(true)
		go startWebServer(ch)
		if !wstate.WaitUntilAvailable(10 * time.Second) {
			fmt.Println("Timed out waiting for server to start")
			os.Exit(1)
		}
	} else {
		ch <- 1
	}

	err := root.Execute()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	<-ch
}

func startWebServer(ch chan int) {
	err := web.Run(wstate)
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprint(os.Stderr, "Web server is running, use Ctrl-C to exit\n")
	if err != nil {
		panic(err)
	}
	ch <- 1
}
