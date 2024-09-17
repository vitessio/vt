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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/vitessio/vitess-tester/src/cmd"
	"os"

	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/test/endtoend/utils"

	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"

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

const (
	defaultKeyspaceName = "mysqltest"
	defaultCellName     = "mysqltest"
)

type hashVindex struct {
	vindexes.Hash
	Type string `json:"type"`
}

func (hv hashVindex) String() string {
	return "xxhash"
}

var (
	vschema vindexes.VSchema

	defaultVschema = vindexes.VSchema{
		Keyspaces: map[string]*vindexes.KeyspaceSchema{
			defaultKeyspaceName: {
				Keyspace: &vindexes.Keyspace{},
				Tables:   map[string]*vindexes.Table{},
				Vindexes: map[string]vindexes.Vindex{
					"xxhash": &hashVindex{Type: "xxhash"},
				},
				Views: map[string]sqlparser.SelectStatement{},
			},
		},
	}
)

func setupCluster() (clusterInstance *cluster.LocalProcessCluster, vtParams, mysqlParams mysql.ConnParams, ksNames []string, close func()) {
	clusterInstance = cluster.NewCluster(defaultCellName, "localhost")

	// Start topo server
	err := clusterInstance.StartTopo()
	if err != nil {
		clusterInstance.Teardown()
		panic(err)
	}

	keyspaces := getKeyspaces()
	for _, keyspace := range keyspaces {
		ksNames = append(ksNames, keyspace.Name)
		vschemaKs, ok := vschema.Keyspaces[keyspace.Name]
		if !ok {
			panic(fmt.Sprintf("keyspace '%s' not found in vschema", keyspace.Name))
		}

		if vschemaKs.Keyspace.Sharded {
			fmt.Printf("starting sharded keyspace: '%s'\n", keyspace.Name)
			err = clusterInstance.StartKeyspace(*keyspace, []string{"-80", "80-"}, 0, false)
			if err != nil {
				clusterInstance.Teardown()
				panic(err.Error())
			}
		} else {
			fmt.Printf("starting unsharded keyspace: '%s'\n", keyspace.Name)
			err = clusterInstance.StartUnshardedKeyspace(*keyspace, 0, false)
			if err != nil {
				clusterInstance.Teardown()
				panic(err.Error())
			}
		}
	}

	// Start vtgate
	err = clusterInstance.StartVtgate()
	if err != nil {
		clusterInstance.Teardown()
		panic(err)
	}

	vtParams = clusterInstance.GetVTParams(ksNames[0])

	// Create the mysqld server we will use to compare the results.
	// We go through all the keyspaces we found in the vschema, and
	// simply create the mysqld process during the first iteration with
	// the first database, following iterations will create new databases.
	var conn *mysql.Conn
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	var closer func()

	for i, keyspace := range keyspaces {
		if i > 0 {
			_, err = conn.ExecuteFetch(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", keyspace.Name), 0, false)
			if err != nil {
				panic(err.Error())
			}
			break
		}

		mysqlParams, closer, err = utils.NewMySQL(clusterInstance, keyspace.Name, "")
		if err != nil {
			clusterInstance.Teardown()
			panic(err)
		}
		conn, err = mysql.Connect(context.Background(), &mysqlParams)
		if err != nil {
			panic(err.Error())
		}
	}

	return clusterInstance, vtParams, mysqlParams, ksNames, func() {
		clusterInstance.Teardown()
		closer()
	}
}

func getKeyspaces() []*cluster.Keyspace {
	ksRaw := cmd.RawKeyspaceVindex{
		Keyspaces: map[string]interface{}{},
	}

	if vschemaFile != "" {
		ksRaw = cmd.ReadVschema(vschemaFile, false)
	} else if vtexplainVschemaFile != "" {
		ksRaw = cmd.ReadVschema(vtexplainVschemaFile, true)
	} else {
		// auto-vschema
		vschema = defaultVschema
		vschema.Keyspaces[defaultKeyspaceName].Keyspace.Sharded = sharded
		ksSchema, err := json.Marshal(vschema.Keyspaces[defaultKeyspaceName])
		if err != nil {
			panic(err.Error())
		}
		ksRaw.Keyspaces[defaultKeyspaceName] = ksSchema
	}

	var keyspaces []*cluster.Keyspace
	var err error
	for key, value := range ksRaw.Keyspaces {
		var ksSchema string
		valueRaw, ok := value.([]uint8)
		if !ok {
			valueRaw, err = json.Marshal(value)
			if err != nil {
				panic(err.Error())
			}
		}
		ksSchema = string(valueRaw)
		keyspaces = append(keyspaces, &cluster.Keyspace{
			Name:    key,
			VSchema: ksSchema,
		})
	}
	return keyspaces
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

	clusterInstance, vtParams, mysqlParams, ksNames, closer := setupCluster()
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
	failed := cmd.ExecuteTests(clusterInstance, vtParams, mysqlParams, tests, reporterSuite, ksNames, vschemaFile, vtexplainVschemaFile, vschema, olap)
	outputFile := reporterSuite.Close()
	if failed {
		log.Errorf("some tests failed 😭\nsee errors in %v", outputFile)
		os.Exit(1)
	}
	println("Great, All tests passed")
}
