package tester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/test/endtoend/utils"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vterrors"

	"github.com/vitessio/vitess-tester/go/data"
)

var _ QueryRunner = (*Tracer)(nil)
var _ QueryRunnerFactory = (*TracerFactory)(nil)

type (
	Tracer struct {
		traceFile            *os.File
		MySQLConn, VtConn    *mysql.Conn
		reporter             Reporter
		inner                QueryRunner
		alreadyWrittenTraces bool
	}
	TracerFactory struct {
		traceFile *os.File
		inner     QueryRunnerFactory
	}
)

func NewTracerFactory(traceFile *os.File, inner QueryRunnerFactory) *TracerFactory {
	return &TracerFactory{
		traceFile: traceFile,
		inner:     inner,
	}
}

func (t *TracerFactory) NewQueryRunner(reporter Reporter, handleCreateTable CreateTableHandler, comparer utils.MySQLCompare) QueryRunner {
	inner := t.inner.NewQueryRunner(reporter, handleCreateTable, comparer)
	return newTracer(t.traceFile, comparer.MySQLConn, comparer.VtConn, reporter, inner)
}

func (t *TracerFactory) Close() {
	_, err := t.traceFile.Write([]byte("]"))
	if err != nil {
		panic(err.Error())
	}
	err = t.traceFile.Close()
	if err != nil {
		panic(err.Error())
	}
}

func newTracer(traceFile *os.File,
	mySQLConn, vtConn *mysql.Conn,
	reporter Reporter,
	inner QueryRunner,
) QueryRunner {
	return &Tracer{
		traceFile: traceFile,
		MySQLConn: mySQLConn,
		VtConn:    vtConn,
		reporter:  reporter,
		inner:     inner,
	}
}

func (t *Tracer) runQuery(q data.Query, expectErr bool, ast sqlparser.Statement) error {
	if sqlparser.IsDMLStatement(ast) && t.traceFile != nil && !expectErr {
		// we don't want to run DMLs twice, so we just run them once while tracing
		var errs []error
		err := t.trace(q)
		if err != nil {
			errs = append(errs, err)
		}

		// we need to run the DMLs on mysql as well
		_, err = t.MySQLConn.ExecuteFetch(q.Query, 10000, false)
		if err != nil {
			errs = append(errs, err)
		}

		return vterrors.Aggregate(errs)
	}

	err := t.inner.runQuery(q, expectErr, ast)
	if err != nil {
		return err
	}

	_, isDDL := ast.(sqlparser.DDLStatement)
	if isDDL {
		// we don't want to trace DDLs
		return nil
	}

	return t.trace(q)
}

// trace writes the query and its trace (fetched from VtConn) as a JSON object into traceFile
func (t *Tracer) trace(query data.Query) error {
	// Marshal the query into JSON format for safe embedding
	queryJSON, err := json.Marshal(query.Query)
	if err != nil {
		return err
	}

	// Fetch the trace for the query using "vexplain trace"
	rs, err := t.VtConn.ExecuteFetch(fmt.Sprintf("vexplain trace %s", query.Query), 10000, false)
	if err != nil {
		return err
	}

	// Extract the trace result and format it with indentation for pretty printing
	var prettyTrace bytes.Buffer
	if err := json.Indent(&prettyTrace, []byte(rs.Rows[0][0].ToString()), "", "  "); err != nil {
		return err
	}

	// Construct the entire JSON entry in memory
	var traceEntry bytes.Buffer
	if t.alreadyWrittenTraces {
		traceEntry.WriteString(",") // Prepend a comma if there are already written traces
	}
	traceEntry.WriteString(fmt.Sprintf(`{"Query": %s, "LineNumber": "%d", "Trace": `, queryJSON, query.Line))
	traceEntry.Write(prettyTrace.Bytes()) // Add the formatted trace
	traceEntry.WriteString("}")           // Close the JSON object

	// Mark that at least one trace has been written
	t.alreadyWrittenTraces = true

	// Write the fully constructed JSON entry to the file
	if _, err := t.traceFile.Write(traceEntry.Bytes()); err != nil {
		return err
	}

	return nil
}
