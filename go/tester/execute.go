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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/test/endtoend/utils"
	vschemapb "vitess.io/vitess/go/vt/proto/vschema"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"
)

type RawKeyspaceVindex struct {
	Keyspaces map[string]interface{} `json:"keyspaces"`
}

const (
	defaultKeyspaceName = "mysqltest"
	defaultCellName     = "mysqltest"
)

func ExecuteTests(
	info ClusterInfo,
	fileNames []string,
	s Suite,
	vschemaFile, vtexplainVschemaFile string,
	olap bool,
	factory QueryRunnerFactory,
) (failed bool) {
	vschemaF := vschemaFile
	if vschemaF == "" {
		vschemaF = vtexplainVschemaFile
	}

	for _, name := range fileNames {
		errReporter := s.NewReporterForFile(name)
		vTester := NewTester(name, errReporter, info, olap, info.vschema, vschemaF, factory)
		err := vTester.Run()
		if err != nil {
			failed = true
			continue
		}
		failed = failed || errReporter.Failed()
		s.CloseReportForFile()
	}

	factory.Close()

	return failed
}

type ClusterInfo struct {
	clusterInstance       *cluster.LocalProcessCluster
	vtParams, mysqlParams mysql.ConnParams
	ksNames               []string
	vschema               *vindexes.VSchema
	closer                func()
}

func SetupCluster(cfg Config) ClusterInfo {
	clusterInstance := cluster.NewCluster(defaultCellName, "localhost")

	errCheck := func(err error) {
		if err == nil {
			return
		}
		clusterInstance.Teardown()
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Start topo server
	err := clusterInstance.StartTopo()
	errCheck(err)

	var ksNames []string
	keyspaces, vschema := getKeyspaces(cfg.VschemaFile, cfg.VtExplainVschemaFile, defaultKeyspaceName, cfg.Sharded)
	for _, keyspace := range keyspaces {
		ksNames = append(ksNames, keyspace.Name)
		vschemaKs, ok := vschema.Keyspaces[keyspace.Name]
		if !ok {
			errCheck(fmt.Errorf("keyspace '%s' not found in vschema", keyspace.Name))
		}

		if vschemaKs.Keyspace.Sharded {
			shardRanges := generateShardRanges(cfg.GetNumberOfShards())
			fmt.Printf("starting sharded keyspace: '%s' with shards %v\n", keyspace.Name, shardRanges)
			err = clusterInstance.StartKeyspace(*keyspace, shardRanges, 0, false)
			errCheck(err)
		} else {
			fmt.Printf("starting unsharded keyspace: '%s'\n", keyspace.Name)
			err = clusterInstance.StartUnshardedKeyspace(*keyspace, 0, false)
			errCheck(err)
		}
	}

	// Start vtgate
	err = clusterInstance.StartVtgate()
	errCheck(err)

	if len(ksNames) == 0 {
		fmt.Println("no keyspaces found in vschema")
		os.Exit(1)
	}

	vtParams := clusterInstance.GetVTParams(ksNames[0])

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

	var closers []func()

	var mysqlParams mysql.ConnParams // TODO: having a single connection is not correct if we are dealing with multiple mysql databases.
	for i, keyspace := range keyspaces {
		if i > 0 {
			_, err = conn.ExecuteFetch(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", keyspace.Name), 0, false)
			errCheck(err)
			break
		}

		var closer func()
		mysqlParams, closer, err = utils.NewMySQL(clusterInstance, keyspace.Name, "")
		if err != nil {
			clusterInstance.Teardown()
			errCheck(err)
		}
		conn, err = mysql.Connect(context.Background(), &mysqlParams)
		if err != nil {
			clusterInstance.Teardown()
			errCheck(err)
		}
		closers = append(closers, closer)
	}

	return ClusterInfo{
		clusterInstance: clusterInstance,
		vtParams:        vtParams,
		mysqlParams:     mysqlParams,
		ksNames:         ksNames,
		vschema:         vschema,
		closer: func() {
			clusterInstance.Teardown()
			for _, closer := range closers {
				closer()
			}
		},
	}
}

func generateShardRanges(numberOfShards int) []string {
	if numberOfShards <= 0 {
		return []string{}
	}

	if numberOfShards == 1 {
		return []string{"-"}
	}

	ranges := make([]string, numberOfShards)
	step := 0x100 / numberOfShards

	for i := range numberOfShards {
		start := i * step
		end := (i + 1) * step

		switch {
		case i == 0:
			ranges[i] = fmt.Sprintf("-%02x", end)
		case i == numberOfShards-1:
			ranges[i] = fmt.Sprintf("%02x-", start)
		default:
			ranges[i] = fmt.Sprintf("%02x-%02x", start, end)
		}
	}

	return ranges
}

func defaultVschema(defaultKeyspaceName string) *vindexes.VSchema {
	return &vindexes.VSchema{
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

func getKeyspaces(vschemaFile, vtexplainVschemaFile, keyspaceName string, sharded bool) (keyspaces []*cluster.Keyspace, vschema *vindexes.VSchema) {
	ksRaw := RawKeyspaceVindex{
		Keyspaces: map[string]interface{}{},
	}
	switch {
	case vschemaFile != "":
		ksRaw, vschema = readVschema(vschemaFile, false)
	case vtexplainVschemaFile != "":
		ksRaw, vschema = readVschema(vtexplainVschemaFile, true)
	default:
		// auto-vschema
		vschema = defaultVschema(keyspaceName)
		vschema.Keyspaces[keyspaceName].Keyspace.Sharded = sharded
		ksSchema, err := json.Marshal(vschema.Keyspaces[keyspaceName])
		exitIf(err, "marshalling vschema")
		ksRaw.Keyspaces[keyspaceName] = ksSchema
	}

	var err error
	for key, value := range ksRaw.Keyspaces {
		var ksSchema string
		valueRaw, ok := value.([]uint8)
		if !ok {
			valueRaw, err = json.Marshal(value)
			exitIf(err, "marshalling keyspace schema")
		}
		ksSchema = string(valueRaw)
		keyspaces = append(keyspaces, &cluster.Keyspace{
			Name:    key,
			VSchema: ksSchema,
		})
	}
	return keyspaces, vschema
}

func readVschema(file string, vtexplain bool) (RawKeyspaceVindex, *vindexes.VSchema) {
	rawVschema, srvVschema := getSrvVschema(file, vtexplain)
	ksRaw, vschema, err := loadVschema(srvVschema, rawVschema)
	exitIf(err, "loading vschema")
	return ksRaw, vschema
}

func getSrvVschema(file string, wrap bool) ([]byte, *vschemapb.SrvVSchema) {
	vschemaStr, err := os.ReadFile(file)
	exitIf(err, "reading vschema file")

	if wrap {
		vschemaStr = []byte(fmt.Sprintf(`{"keyspaces": %s}`, vschemaStr))
	}

	var srvVSchema vschemapb.SrvVSchema
	err = json.Unmarshal(vschemaStr, &srvVSchema)
	exitIf(err, "unmarshalling vschema")

	if len(srvVSchema.Keyspaces) == 0 {
		exitIf(errors.New("no keyspaces found"), "loading vschema")
	}

	return vschemaStr, &srvVSchema
}

func loadVschema(srvVschema *vschemapb.SrvVSchema, rawVschema []byte) (RawKeyspaceVindex, *vindexes.VSchema, error) {
	vschema := vindexes.BuildVSchema(srvVschema, sqlparser.NewTestParser())
	if len(vschema.Keyspaces) == 0 {
		return RawKeyspaceVindex{}, nil, errors.New("no keyspace defined in vschema")
	}

	var rkv RawKeyspaceVindex
	err := json.Unmarshal(rawVschema, &rkv)
	if err != nil {
		return RawKeyspaceVindex{}, nil, err
	}
	return rkv, vschema, nil
}

type hashVindex struct {
	vindexes.Hash
	Type string `json:"type"`
}

func (hv hashVindex) String() string {
	return "xxhash"
}
