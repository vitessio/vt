// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/vitessio/vitess-tester/src/cmd"
	vitess_tester "github.com/vitessio/vitess-tester/src/vitess-tester"
)

var (
	logLevel             string
	sharded              bool
	olap                 bool
	vschemaFile          string
	vtexplainVschemaFile string
	xunit                bool
)

func init() {
	flag.BoolVar(&olap, "olap", false, "Use OLAP to run the queries.")
	flag.StringVar(&logLevel, "log-level", "error", "The log level of vitess-tester: info, warn, error, debug.")
	flag.BoolVar(&xunit, "xunit", false, "Get output in an xml file instead of errors directory")

	flag.BoolVar(&sharded, "sharded", false, "Run all tests on a sharded keyspace and using auto-vschema. This cannot be used with either -vschema or -vtexplain-vschema.")
	flag.StringVar(&vschemaFile, "vschema", "", "Disable auto-vschema by providing your own vschema file. This cannot be used with either -vtexplain-vschema or -sharded.")
	flag.StringVar(&vtexplainVschemaFile, "vtexplain-vschema", "", "Disable auto-vschema by providing your own vtexplain vschema file. This cannot be used with either -vschema or -sharded.")
}

func main() {
	flag.Parse()
	tests := flag.Args()

	err := vitess_tester.CheckEnvironment()
	if err != nil {
		fmt.Println("Fatal error:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	a := vschemaFile != ""
	b := vtexplainVschemaFile != ""
	if a && b || a && sharded || b && sharded {
		log.Errorf("specify only one of the following flags: -vschema, -vtexplain-vschema, -sharded")
		os.Exit(1)
	}

	if ll := os.Getenv("LOG_LEVEL"); ll != "" {
		logLevel = ll
	}
	if logLevel != "" {
		ll, err := log.ParseLevel(logLevel)
		if err != nil {
			log.Errorf("error parsing log level %s: %v", logLevel, err)
		}
		log.SetLevel(ll)
	}

	if len(tests) == 0 {
		log.Errorf("no tests specified")
		os.Exit(1)
	}

	log.Infof("running tests: %v", tests)

	clusterInstance, vtParams, mysqlParams, ksNames, closer := cmd.SetupCluster(vschemaFile, vtexplainVschemaFile, sharded)
	defer closer()

	// remove errors folder if exists
	err = os.RemoveAll("errors")
	if err != nil {
		panic(err.Error())
	}

	var reporterSuite vitess_tester.Suite
	if xunit {
		reporterSuite = vitess_tester.NewXMLTestSuite()
	} else {
		reporterSuite = vitess_tester.NewFileReporterSuite()
	}
	failed := cmd.ExecuteTests(clusterInstance, vtParams, mysqlParams, tests, reporterSuite, ksNames, vschemaFile, vtexplainVschemaFile, olap)
	outputFile := reporterSuite.Close()
	if failed {
		log.Errorf("some tests failed 😭\nsee errors in %v", outputFile)
		os.Exit(1)
	}
	println("Great, All tests passed")
}
