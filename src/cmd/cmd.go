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
	"context"
	"encoding/json"
	"fmt"
	"os"

	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/test/endtoend/utils"
	vschemapb "vitess.io/vitess/go/vt/proto/vschema"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"

	vitess_tester "github.com/vitessio/vitess-tester/src/vitess-tester"
)

type RawKeyspaceVindex struct {
	Keyspaces map[string]interface{} `json:"keyspaces"`
}

var (
	vschema vindexes.VSchema
)

const (
	defaultKeyspaceName = "mysqltest"
	defaultCellName     = "mysqltest"
)

func ExecuteTests(
	clusterInstance *cluster.LocalProcessCluster,
	vtParams, mysqlParams mysql.ConnParams,
	fileNames []string,
	s vitess_tester.Suite,
	ksNames []string,
	vschemaFile, vtexplainVschemaFile string,
	olap bool,
) (failed bool) {
	vschemaF := vschemaFile
	if vschemaF == "" {
		vschemaF = vtexplainVschemaFile
	}
	for _, name := range fileNames {
		errReporter := s.NewReporterForFile(name)
		vTester := vitess_tester.NewTester(name, errReporter, clusterInstance, vtParams, mysqlParams, olap, ksNames, vschema, vschemaF)
		err := vTester.Run()
		if err != nil {
			failed = true
			continue
		}
		failed = failed || errReporter.Failed()
		s.CloseReportForFile()
	}
	return
}

func SetupCluster(
	vschemaFile, vtexplainVschemaFile string,
	sharded bool,
) (clusterInstance *cluster.LocalProcessCluster, vtParams, mysqlParams mysql.ConnParams, ksNames []string, close func()) {
	clusterInstance = cluster.NewCluster(defaultCellName, "localhost")

	// Start topo server
	err := clusterInstance.StartTopo()
	if err != nil {
		clusterInstance.Teardown()
		panic(err)
	}

	keyspaces := getKeyspaces(vschemaFile, vtexplainVschemaFile, defaultKeyspaceName, sharded)
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

func defaultVschema(defaultKeyspaceName string) vindexes.VSchema {
	return vindexes.VSchema{
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
}

func getKeyspaces(vschemaFile, vtexplainVschemaFile, keyspaceName string, sharded bool) (keyspaces []*cluster.Keyspace) {
	ksRaw := RawKeyspaceVindex{
		Keyspaces: map[string]interface{}{},
	}

	if vschemaFile != "" {
		ksRaw = readVschema(vschemaFile, false)
	} else if vtexplainVschemaFile != "" {
		ksRaw = readVschema(vtexplainVschemaFile, true)
	} else {
		// auto-vschema
		vschema = defaultVschema(keyspaceName)
		vschema.Keyspaces[keyspaceName].Keyspace.Sharded = sharded
		ksSchema, err := json.Marshal(vschema.Keyspaces[keyspaceName])
		if err != nil {
			panic(err.Error())
		}
		ksRaw.Keyspaces[keyspaceName] = ksSchema
	}

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

func readVschema(file string, vtexplain bool) RawKeyspaceVindex {
	rawVschema, srvVschema, err := getSrvVschema(file, vtexplain)
	if err != nil {
		panic(err.Error())
	}
	ksRaw, err := loadVschema(srvVschema, rawVschema)
	if err != nil {
		panic(err.Error())
	}
	return ksRaw
}

func getSrvVschema(file string, wrap bool) ([]byte, *vschemapb.SrvVSchema, error) {
	vschemaStr, err := os.ReadFile(file)
	if err != nil {
		panic(err.Error())
	}

	if wrap {
		vschemaStr = []byte(fmt.Sprintf(`{"keyspaces": %s}`, vschemaStr))
	}

	var srvVSchema vschemapb.SrvVSchema
	err = json.Unmarshal(vschemaStr, &srvVSchema)
	if err != nil {
		return nil, nil, err
	}

	if len(srvVSchema.Keyspaces) == 0 {
		return nil, nil, fmt.Errorf("no keyspaces found")
	}

	return vschemaStr, &srvVSchema, nil
}

func loadVschema(srvVschema *vschemapb.SrvVSchema, rawVschema []byte) (rkv RawKeyspaceVindex, err error) {
	vschema := *(vindexes.BuildVSchema(srvVschema, sqlparser.NewTestParser()))
	if len(vschema.Keyspaces) == 0 {
		err = fmt.Errorf("no keyspace defined in vschema")
		return
	}

	var rk RawKeyspaceVindex
	err = json.Unmarshal(rawVschema, &rk)
	return
}

type hashVindex struct {
	vindexes.Hash
	Type string `json:"type"`
}

func (hv hashVindex) String() string {
	return "xxhash"
}
