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

package summarize

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
)

type (
	node struct {
		ID       string `json:"id"`
		RowCount int    `json:"size"`
	}

	link struct {
		Source     string   `json:"source"`
		SourceIdx  int      `json:"source_idx"`
		Target     string   `json:"target"`
		TargetIdx  int      `json:"target_idx"`
		Value      int      `json:"value"`
		Type       string   `json:"type"`
		Curvature  float64  `json:"curvature"`
		Predicates []string `json:"predicates"`
	}

	graphData struct {
		Nodes []node `json:"nodes"`
		Links []link `json:"links"`
	}

	forceGraphData struct {
		maxValue   int
		maxNumRows int
		graphData
	}
)

func createForceGraphData(s *Summary) forceGraphData {
	result := &forceGraphData{}

	idxTableNode := make(map[string]int)
	result.maxNumRows = 0
	for _, table := range s.Tables {
		result.Nodes = append(result.Nodes, node{ID: table.Table, RowCount: table.RowCount})
		idxTableNode[table.Table] = len(result.Nodes) - 1
		if table.RowCount > result.maxNumRows {
			result.maxNumRows = table.RowCount
		}
	}

	addJoins(s, result, idxTableNode)

	addTransactions(s, result, idxTableNode)

	addForeignKeys(s, result, idxTableNode)

	m := make(map[graphKey][]int)

	for i, l := range result.Links {
		if l.Value > result.maxValue {
			result.maxValue = l.Value
		}
		m[createGraphKey(l.Source, l.Target)] = append(m[createGraphKey(l.Source, l.Target)], i)
	}
	const curvatureMinMax = 0.5
	for _, links := range m {
		if len(links) == 1 {
			continue
		}
		delta := 2 * curvatureMinMax / (len(links) - 1)
		for i, idx := range links {
			result.Links[idx].Curvature = -curvatureMinMax + float64(i*delta)
		}
	}

	return *result
}

func addForeignKeys(s *Summary, result *forceGraphData, idxTableNode map[string]int) {
	for _, ts := range s.Tables {
		for _, fk := range ts.ReferencedTables {
			if t := s.GetTable(ts.Table); t == nil {
				s.AddTable(&TableSummary{Table: ts.Table})
			}
			if t := s.GetTable(fk.ReferencedTableName); t == nil {
				s.AddTable(&TableSummary{Table: fk.ReferencedTableName})
			}
			result.Links = append(result.Links, link{
				Source:    ts.Table,
				SourceIdx: idxTableNode[ts.Table],
				Target:    fk.ReferencedTableName,
				TargetIdx: idxTableNode[fk.ReferencedTableName],
				Value:     1,
				Type:      "fk",
			})
		}
	}
}

func addTransactions(s *Summary, result *forceGraphData, idxTableNode map[string]int) {
	txTablesMap := make(map[graphKey]int)
	for _, transaction := range s.Transactions {
		var tables []string
		for _, query := range transaction.Queries {
			tables = append(tables, query.Table)
		}
		tables = uniquefy(tables)

		for i, ti := range tables {
			for j, tj := range tables {
				if j <= i {
					continue
				}
				txTablesMap[createGraphKey(ti, tj)]++
			}
		}
	}
	for key, val := range txTablesMap {
		result.Links = append(result.Links, link{
			Source:    key.Tbl1,
			SourceIdx: idxTableNode[key.Tbl1],
			Target:    key.Tbl2,
			TargetIdx: idxTableNode[key.Tbl2],
			Value:     val,
			Type:      "tx",
		})
	}
}

func addJoins(s *Summary, result *forceGraphData, idxTableNode map[string]int) {
	for _, join := range s.Joins {
		var preds []string
		for _, predicate := range join.predicates {
			preds = append(preds, predicate.String())
		}
		result.Links = append(result.Links, link{
			Source:     join.Tbl1,
			SourceIdx:  idxTableNode[join.Tbl1],
			Target:     join.Tbl2,
			TargetIdx:  idxTableNode[join.Tbl2],
			Value:      join.Occurrences,
			Type:       "join",
			Predicates: preds,
		})
	}
}

func createGraphKey(tableA, tableB string) graphKey {
	if tableA < tableB {
		return graphKey{Tbl1: tableA, Tbl2: tableB}
	}
	return graphKey{Tbl1: tableB, Tbl2: tableA}
}

func renderQueryGraph(s *Summary) error {
	data := createForceGraphData(s)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("could not create a listener: %w", err)
	}
	defer listener.Close()

	// Get the assigned port
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return errors.New("could not create a listener")
	}
	fmt.Printf("Server started at http://localhost:%d\nExit the program with CTRL+C\n", addr.Port)

	// Start the server
	// nolint: gosec,nolintlint // this is all ran locally so no need to care about vulnerabilities around timeouts
	return http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		err := serveIndex(w, data)
		if err != nil {
			fmt.Println(err.Error())
		}
	}))
}

//go:embed graph-template.gohtml
var templateHTML string

// Function to dynamically generate and serve index.html
func serveIndex(w http.ResponseWriter, data forceGraphData) error {
	dataBytes, err := json.Marshal(data.graphData)
	if err != nil {
		return fmt.Errorf("could not marshal graphData: %w", err)
	}

	tmpl, err := template.New("index").Parse(templateHTML)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return err
	}

	d := struct {
		Data       any
		MaxValue   int
		MaxNumRows int
	}{
		// nolint: gosec,nolintlint // this is all ran locally so no need to care about vulnerabilities around escaping
		Data:       template.JS(dataBytes),
		MaxValue:   data.maxValue,
		MaxNumRows: data.maxNumRows,
	}

	if err := tmpl.Execute(w, d); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		return err
	}

	return nil
}
