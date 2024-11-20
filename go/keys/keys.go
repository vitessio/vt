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

package keys

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	querypb "vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/planbuilder/operators"
	"vitess.io/vitess/go/vt/vtgate/planbuilder/plancontext"
	"vitess.io/vitess/go/vt/vtgate/semantics"

	"github.com/vitessio/vt/go/data"
)

type (
	Config struct {
		FileName string
		Loader   data.Loader
	}
	// Output represents the output generated by 'vt keys'
	Output struct {
		Queries []QueryAnalysisResult `json:"queries"`
		Failed  []QueryFailedResult   `json:"failed,omitempty"`
	}
	queryList struct {
		queries map[string]*QueryAnalysisResult
		failed  map[string]*QueryFailedResult
	}
	// QueryAnalysisResult represents the result of analyzing a query in a query log. It contains the query structure, the number of
	// times the query was used, the line numbers where the query was used, the table name, grouping columns, join columns,
	// filter columns, and the statement type.
	QueryAnalysisResult struct {
		QueryStructure  string                    `json:"queryStructure"`
		UsageCount      int                       `json:"usageCount"`
		LineNumbers     []int                     `json:"lineNumbers"`
		TableNames      []string                  `json:"tableNames,omitempty"`
		GroupingColumns []operators.Column        `json:"groupingColumns,omitempty"`
		JoinPredicates  []operators.JoinPredicate `json:"joinPredicates,omitempty"`
		FilterColumns   []operators.ColumnUse     `json:"filterColumns,omitempty"`
		StatementType   string                    `json:"statementType"`
		QueryTime       float64                   `json:"queryTime,omitempty"`
		LockTime        float64                   `json:"lockTime,omitempty"`
		RowsSent        int                       `json:"rowsSent,omitempty"`
		RowsExamined    int                       `json:"rowsExamined,omitempty"`
		Timestamp       int64                     `json:"timestamp,omitempty"`
	}
	QueryFailedResult struct {
		Query       string `json:"query"`
		LineNumbers []int  `json:"lineNumbers"`
		Error       string `json:"error"`
	}
)

func Run(cfg Config) error {
	return run(os.Stdout, cfg)
}

func run(out io.Writer, cfg Config) error {
	si := &SchemaInfo{
		Tables: make(map[string]Columns),
	}
	ql := &queryList{
		queries: make(map[string]*QueryAnalysisResult),
		failed:  make(map[string]*QueryFailedResult),
	}

	loader := cfg.Loader.Load(cfg.FileName)

	_ = data.ForeachSQLQuery(loader, func(query data.Query) error {
		process(query, si, ql)
		return nil
	})

	closeErr := loader.Close()
	jsonWriteErr := ql.writeJSONTo(out)

	return errors.Join(closeErr, jsonWriteErr)
}

func process(q data.Query, si *SchemaInfo, ql *queryList) {
	ast, bv, err := sqlparser.NewTestParser().Parse2(q.Query)
	if err != nil {
		ql.addFailedQuery(q, err)
		return
	}

	switch ast := ast.(type) {
	case *sqlparser.CreateTable:
		si.handleCreateTable(ast)
	case sqlparser.Statement:
		ql.processQuery(si, ast, q, bv)
	}
}

func (ql *queryList) processQuery(si *SchemaInfo, ast sqlparser.Statement, q data.Query, bv sqlparser.BindVars) {
	// handle panics
	defer func() {
		if r := recover(); r != nil {
			ql.addFailedQuery(q, fmt.Errorf("panic: %v", r))
		}
	}()

	mapBv := make(map[string]*querypb.BindVariable)
	reservedVars := sqlparser.NewReservedVars("", bv)
	err := sqlparser.Normalize(ast, reservedVars, mapBv)
	if err != nil {
		ql.addFailedQuery(q, err)
		return
	}

	st, err := semantics.Analyze(ast, "ks", si)
	if err != nil {
		ql.addFailedQuery(q, err)
		return
	}
	ctx := &plancontext.PlanningContext{
		ReservedVars: reservedVars,
		SemTable:     st,
	}

	structure := sqlparser.CanonicalString(ast)
	r, found := ql.queries[structure]
	if found {
		r.UsageCount++
		r.LineNumbers = append(r.LineNumbers, q.Line)
		r.QueryTime += q.QueryTime
		r.LockTime += q.LockTime
		r.RowsSent += q.RowsSent
		r.RowsExamined += q.RowsExamined
		return
	}

	var tableNames []string
	for _, t := range ctx.SemTable.Tables {
		rtbl, ok := t.(*semantics.RealTable)
		if !ok || rtbl.Table == nil {
			continue
		}
		tableNames = append(tableNames, rtbl.Table.Name.String())
	}

	result := operators.GetVExplainKeys(ctx, ast)
	ql.queries[structure] = &QueryAnalysisResult{
		QueryStructure:  structure,
		StatementType:   result.StatementType,
		UsageCount:      1,
		LineNumbers:     []int{q.Line},
		TableNames:      tableNames,
		GroupingColumns: result.GroupingColumns,
		JoinPredicates:  result.JoinPredicates,
		FilterColumns:   result.FilterColumns,
		QueryTime:       q.QueryTime,
		LockTime:        q.LockTime,
		RowsSent:        q.RowsSent,
		RowsExamined:    q.RowsExamined,
		Timestamp:       q.Timestamp,
	}
}

func (ql *queryList) addFailedQuery(q data.Query, err error) {
	key := q.Query + err.Error()
	if v, exists := ql.failed[key]; exists {
		v.LineNumbers = append(v.LineNumbers, q.Line)
	} else {
		ql.failed[key] = &QueryFailedResult{
			Query:       q.Query,
			LineNumbers: []int{q.Line},
			Error:       err.Error(),
		}
	}
}

// writeJsonTo writes the query list, sorted by the first line number of the query, to the given writer.
func (ql *queryList) writeJSONTo(w io.Writer) error {
	values := make([]QueryAnalysisResult, 0, len(ql.queries))
	for _, result := range ql.queries {
		values = append(values, *result)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].LineNumbers[0] < values[j].LineNumbers[0]
	})

	failedQueries := make([]QueryFailedResult, 0, len(ql.failed))
	for _, result := range ql.failed {
		failedQueries = append(failedQueries, *result)
	}
	sort.Slice(failedQueries, func(i, j int) bool {
		return failedQueries[i].LineNumbers[0] < failedQueries[j].LineNumbers[0]
	})

	res := Output{
		Queries: values,
		Failed:  failedQueries,
	}

	jsonData, err := json.MarshalIndent(res, "  ", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(jsonData)
	if err != nil {
		return err
	}

	return err
}
