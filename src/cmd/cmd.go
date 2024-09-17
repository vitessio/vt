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
	"encoding/json"
	"fmt"
	vitess_tester "github.com/vitessio/vitess-tester/src/vitess-tester"
	"os"
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/cluster"
	vschemapb "vitess.io/vitess/go/vt/proto/vschema"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"
)

type RawKeyspaceVindex struct {
	Keyspaces map[string]interface{} `json:"keyspaces"`
}

func ExecuteTests(
	clusterInstance *cluster.LocalProcessCluster,
	vtParams, mysqlParams mysql.ConnParams,
	fileNames []string,
	s vitess_tester.Suite,
	ksNames []string,
	vschemaFile, vtexplainVschemaFile string,
	vschema vindexes.VSchema,
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

func ReadVschema(file string, vtexplain bool) RawKeyspaceVindex {
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
