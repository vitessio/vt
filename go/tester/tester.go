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
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/test/endtoend/utils"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"

	"github.com/vitessio/vt/go/data"
	"github.com/vitessio/vt/go/tester/state"
	"github.com/vitessio/vt/go/typ"
)

type (
	Tester struct {
		name string

		clusterInstance *cluster.LocalProcessCluster
		vtParams        mysql.ConnParams
		mysqlParams     *mysql.ConnParams

		// connections

		MySQLConn, VtConn *mysql.Conn

		olap        bool
		ksNames     []string
		vschema     *vindexes.VSchema
		vschemaFile string
		vexplain    string

		state *state.State

		reporter Reporter

		qr QueryRunner
	}

	QueryRunner interface {
		runQuery(q data.Query, ast sqlparser.Statement, state *state.State) error
	}

	QueryRunnerFactory interface {
		NewQueryRunner(reporter Reporter, handleCreateTable CreateTableHandler, comparer utils.MySQLCompare, cluster *cluster.LocalProcessCluster, vschema *vindexes.VSchema) QueryRunner
		Close()
	}
)

func NewTester(name string, reporter Reporter, info ClusterInfo, olap bool, vschema *vindexes.VSchema, vschemaFile string, factory QueryRunnerFactory) *Tester {
	t := &Tester{
		name:            name,
		reporter:        reporter,
		vtParams:        info.vtParams,
		mysqlParams:     info.mysqlParams,
		clusterInstance: info.clusterInstance,
		ksNames:         info.ksNames,
		vschema:         vschema,
		vschemaFile:     vschemaFile,
		olap:            olap,
		state:           state.NewState(utils.BinaryIsAtLeastAtVersion),
	}

	var mcmp utils.MySQLCompare
	var err error
	if t.mysqlParams != nil {
		mcmp, err = utils.NewMySQLCompare(t.reporter, t.vtParams, *t.mysqlParams)
		exitIf(err, "creating MySQLCompare")
	} else {
		vtConn, err := mysql.Connect(context.Background(), &t.vtParams)
		exitIf(err, "connecting to MySQL")
		mcmp = utils.MySQLCompare{VtConn: vtConn}
	}
	createTableHandler := t.handleCreateTable
	if !t.autoVSchema() {
		createTableHandler = func(*sqlparser.CreateTable) func() { return func() {} }
	}
	t.qr = factory.NewQueryRunner(reporter, createTableHandler, mcmp, info.clusterInstance, vschema)

	return t
}

func (t *Tester) preProcess() {
	if t.olap {
		_, err := t.VtConn.ExecuteFetch("set workload = 'olap'", 0, false)
		exitIf(err, "setting workload to olap by executing query")
	}
}

func (t *Tester) postProcess() error {
	r, err := t.MySQLConn.ExecuteFetch("show tables", 1000, true)
	if err != nil {
		return fmt.Errorf("running show tables: %w", err)
	}
	for _, row := range r.Rows {
		_, err := t.VtConn.ExecuteFetch(fmt.Sprintf("drop table %s", row[0].ToString()), 100, false)
		if err != nil {
			return fmt.Errorf("dropping table %s: %w", row[0].ToString(), err)
		}
		if t.MySQLConn != nil {
			_, err := t.MySQLConn.ExecuteFetch(fmt.Sprintf("drop table %s", row[0].ToString()), 100, false)
			if err != nil {
				return fmt.Errorf("dropping table %s: %w", row[0].ToString(), err)
			}
		}
	}
	t.VtConn.Close()
	if t.MySQLConn != nil {
		t.MySQLConn.Close()
	}
	return nil
}

const PERM os.FileMode = 0o755

func (t *Tester) runVexplain(q string) {
	result, err := t.VtConn.ExecuteFetch(fmt.Sprintf("vexplain %s %s", t.vexplain, q), -1, false)
	t.vexplain = ""
	if err != nil {
		t.reporter.AddFailure(err)
	}

	t.reporter.AddInfo(fmt.Sprintf("VExplain Output:\n %s\n", result.Rows[0][0].ToString()))
}

func (t *Tester) skipIfBelow(q string) {
	strs := strings.Split(q, " ")
	if len(strs) != 3 {
		t.reporter.AddFailure(fmt.Errorf("incorrect syntax for typ.Q_SKIP_IF_BELOW_VERSION in: %v", q))
		return
	}
	v, err := strconv.Atoi(strs[2])
	if err != nil {
		t.reporter.AddFailure(err)
		return
	}
	err = t.state.SetSkipBelowVersion(strs[1], v)
	if err != nil {
		t.reporter.AddFailure(err)
	}
}

func (t *Tester) prepareVExplain(q string) {
	strs := strings.Split(q, " ")
	if len(strs) != 2 {
		t.reporter.AddFailure(fmt.Errorf("incorrect syntax for typ.VExplain in: %v", q))
		return
	}

	t.vexplain = strs[1]
}

func (t *Tester) handleQuery(q data.Query) {
	var err error
	switch q.Type {
	case typ.Skip:
		err = t.state.SetSkipNext()
	case typ.SkipIfBelowVersion:
		t.skipIfBelow(q.Query)
	case typ.Error:
		err = t.state.SetErrorExpected()
	case typ.VExplain:
		t.prepareVExplain(q.Query)
	case typ.WaitForAuthoritative:
		t.waitAuthoritative(q.Query)
	case typ.Query:
		if t.vexplain == "" {
			t.runQuery(q)
			return
		}
		t.runVexplain(q.Query)
	case typ.VitessOnly:
		err = vitessOrMySQLOnly(q.Query, t.state.BeginVitessOnly, t.state.EndVitessOnly)
	case typ.MysqlOnly:
		err = vitessOrMySQLOnly(q.Query, t.state.BeginMySQLOnly, t.state.EndMySQLOnly)
	case typ.Reference:
		err = t.state.SetReference()
	default:
		t.reporter.AddFailure(fmt.Errorf("%s not supported", q.Type.String()))
	}
	if err != nil {
		t.reporter.AddFailure(err)
	}
}

func (t *Tester) Run() (err error) {
	t.preProcess()
	if t.autoVSchema() {
		defer func() {
			postErr := t.postProcess()
			if postErr == nil {
				return
			}
			if err == nil {
				err = postErr
			}
			err = errors.Join(err, postErr)
		}()
	}
	queries, err := data.LoadQueries(t.name)
	if err != nil {
		t.reporter.AddFailure(err)
		return err
	}

	for _, q := range queries {
		t.handleQuery(q)
	}
	fmt.Printf("%s\n", t.reporter.Report())

	return nil
}

func vitessOrMySQLOnly(query string, begin, end func() error) error {
	strs := strings.Split(query, " ")
	if len(strs) != 2 {
		return fmt.Errorf("incorrect syntax in: %v", query)
	}

	switch strs[1] {
	case "begin":
		return begin()
	case "end":
		return end()
	default:
		return fmt.Errorf("incorrect syntax in: %v", query)
	}
}

func (t *Tester) runQuery(q data.Query) {
	if t.state.ShouldSkip() {
		return
	}
	t.reporter.AddTestCase(q.Query, q.Line)
	parser := sqlparser.NewTestParser()
	ast, err := parser.Parse(q.Query)
	if err != nil {
		t.reporter.AddFailure(err)
		return
	}
	err = t.qr.runQuery(q, ast, t.state)
	if err != nil {
		t.reporter.AddFailure(err)
	}
	t.reporter.EndTestCase()
}

func (t *Tester) findTable(name string) (ks string, err error) {
	for ksName, ksSchema := range t.vschema.Keyspaces {
		for _, table := range ksSchema.Tables {
			if table.Name.String() == name {
				if ks != "" {
					return "", fmt.Errorf("table %s found in multiple keyspaces", name)
				}
				ks = ksName
			}
		}
	}
	if ks == "" {
		return "", fmt.Errorf("table %s not found in any keyspace", name)
	}
	return ks, nil
}

func (t *Tester) waitAuthoritative(query string) {
	var tblName, ksName string
	strs := strings.Split(query, " ")
	switch len(strs) {
	case 2:
		tblName = strs[1]
		var err error
		ksName, err = t.findTable(tblName)
		if err != nil {
			t.reporter.AddFailure(err)
			return
		}
	case 3:
		tblName = strs[1]
		ksName = strs[2]

	default:
		t.reporter.AddFailure(fmt.Errorf("expected table name and keyspace for wait_authoritative in: %v", query))
	}

	log.Infof("Waiting for authoritative schema for table %s", tblName)
	err := utils.WaitForAuthoritative(t.reporter, ksName, tblName, t.clusterInstance.VtgateProcess.ReadVSchema)
	if err != nil {
		t.reporter.AddFailure(fmt.Errorf("failed to wait for authoritative schema for table %s: %v", tblName, err))
	}
}

func newPrimaryKeyIndexDefinitionSingleColumn(name sqlparser.IdentifierCI) *sqlparser.IndexDefinition {
	index := &sqlparser.IndexDefinition{
		Info: &sqlparser.IndexInfo{
			Name: sqlparser.NewIdentifierCI("PRIMARY"),
			Type: sqlparser.IndexTypePrimary,
		},
		Columns: []*sqlparser.IndexColumn{{Column: name}},
	}
	return index
}

func (t *Tester) autoVSchema() bool {
	return t.vschemaFile == ""
}

func getShardingKeysForTable(create *sqlparser.CreateTable) (sks []sqlparser.IdentifierCI) {
	var allIDCI []sqlparser.IdentifierCI
	// first we normalize the primary keys
	for _, col := range create.TableSpec.Columns {
		if col.Type.Options.KeyOpt == sqlparser.ColKeyPrimary {
			create.TableSpec.Indexes = append(create.TableSpec.Indexes, newPrimaryKeyIndexDefinitionSingleColumn(col.Name))
			col.Type.Options.KeyOpt = sqlparser.ColKeyNone
		}
		allIDCI = append(allIDCI, col.Name)
	}

	// and now we can fetch the primary keys
	for _, index := range create.TableSpec.Indexes {
		if index.Info.Type == sqlparser.IndexTypePrimary {
			for _, column := range index.Columns {
				sks = append(sks, column.Column)
			}
		}
	}

	// if we have no primary keys, we'll use all columns as the sharding keys
	if len(sks) == 0 {
		sks = allIDCI
	}
	return
}

func (t *Tester) handleCreateTable(create *sqlparser.CreateTable) func() {
	sks := getShardingKeysForTable(create)

	shardingKeys := &vindexes.ColumnVindex{
		Columns: sks,
		Name:    "xxhash",
		Type:    "xxhash",
	}

	ks := t.vschema.Keyspaces[t.ksNames[0]]
	tableName := create.Table.Name
	ks.Tables[tableName.String()] = &vindexes.Table{
		Name:           tableName,
		Keyspace:       ks.Keyspace,
		ColumnVindexes: []*vindexes.ColumnVindex{shardingKeys},
	}

	ksJSON, err := json.Marshal(ks)
	exitIf(err, "marshalling keyspace schema")

	err = t.clusterInstance.VtctldClientProcess.ApplyVSchema(t.ksNames[0], string(ksJSON))
	exitIf(err, "applying vschema")

	return func() {
		err := utils.WaitForAuthoritative(t.reporter, t.ksNames[0], create.Table.Name.String(), t.clusterInstance.VtgateProcess.ReadVSchema)
		exitIf(err, "waiting for authoritative schema after auto-vschema update ")
	}
}
