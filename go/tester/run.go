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

package tester

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/test/endtoend/cluster"
)

type Config struct {
	LogLevel             string
	OLAP                 bool
	Sharded              bool
	XUnit                bool
	VschemaFile          string
	VtExplainVschemaFile string
	TraceFile            string
	Tests                []string
	NumberOfShards       int
	Compare              bool
}

func (cfg Config) GetNumberOfShards() int {
	if cfg.NumberOfShards == 0 {
		return 2
	}
	return cfg.NumberOfShards
}

func Run(cfg Config) error {
	err := CheckEnvironment()
	if err != nil {
		return fmt.Errorf("error reading environment variables: %w", err)
	}

	a := cfg.VschemaFile != ""
	b := cfg.VtExplainVschemaFile != ""
	if a && b || a && cfg.Sharded || b && cfg.Sharded {
		return errors.New("specify only one of the following flags: -vschema, -vtexplain-vschema, -sharded")
	}

	if cfg.NumberOfShards > 0 && !(cfg.Sharded || cfg.VschemaFile != "" || cfg.VtExplainVschemaFile != "") {
		return errors.New("number-of-shards can only be used with -sharded, -vschema or -vtexplain-vschema")
	}

	if ll := os.Getenv("LOG_LEVEL"); ll != "" {
		cfg.LogLevel = ll
	}
	if cfg.LogLevel != "" {
		ll, err := log.ParseLevel(cfg.LogLevel)
		if err != nil {
			log.Errorf("error parsing log level %s: %v", cfg.LogLevel, err)
		}
		log.SetLevel(ll)
	}

	if len(cfg.Tests) == 0 {
		return errors.New("no tests specified")
	}

	log.Infof("running tests: %v", cfg.Tests)

	clusterInfo := SetupCluster(cfg)
	defer clusterInfo.closer()

	// remove errors folder if exists
	err = os.RemoveAll("errors")
	if err != nil {
		return fmt.Errorf("removing errors folder: %w", err)
	}

	var reporterSuite Suite
	if cfg.XUnit {
		reporterSuite = NewXMLTestSuite()
	} else {
		reporterSuite = NewFileReporterSuite(getVschema(clusterInfo.clusterInstance))
	}
	failed := ExecuteTests(clusterInfo, cfg.Tests, reporterSuite, cfg.VschemaFile, cfg.VtExplainVschemaFile, cfg.OLAP, getQueryRunnerFactory(cfg.TraceFile))
	outputFile := reporterSuite.Close()
	if failed {
		return fmt.Errorf("some tests failed 😭\nsee errors in %v", outputFile)
	}
	println("Great, All tests passed")
	return nil
}

func getQueryRunnerFactory(traceFile string) QueryRunnerFactory {
	inner := ComparingQueryRunnerFactory{}
	if traceFile == "" {
		return inner
	}

	var err error
	writer, err := os.Create(traceFile)
	exitIf(err, "creating trace file")

	_, err = writer.Write([]byte("["))
	exitIf(err, "writing to trace file")
	return NewTracerFactory(writer, inner)
}

func getVschema(clusterInstance *cluster.LocalProcessCluster) func() []byte {
	return func() []byte {
		httpClient := &http.Client{Timeout: 5 * time.Second}
		resp, err := httpClient.Get(clusterInstance.VtgateProcess.VSchemaURL)
		if err != nil {
			log.Error(err.Error())
			return nil
		}
		defer resp.Body.Close()
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error(err.Error())
			return nil
		}

		return res
	}
}

func exitIf(err error, message string) {
	if err == nil {
		return
	}
	c := color.New(color.FgRed)
	_, _ = c.Fprintf(os.Stderr, "%s: %s\n", message, err.Error())
	os.Exit(1)
}
