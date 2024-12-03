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
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
)

type (
	node struct {
		ID string `json:"id"`
	}

	link struct {
		Source    string `json:"source"`
		SourceIdx int    `json:"source_idx"`
		Target    string `json:"target"`
		TargetIdx int    `json:"target_idx"`
		Value     int    `json:"value"`
		Type      string `json:"type"`
	}

	forceGraphData struct {
		Nodes []node `json:"nodes"`
		Links []link `json:"links"`
	}
)

func createForceGraphData(s *Summary) forceGraphData {
	var data forceGraphData

	idxTableNode := make(map[string]int)
	for _, table := range s.tables {
		data.Nodes = append(data.Nodes, node{ID: table.Table})
		idxTableNode[table.Table] = len(data.Nodes) - 1
	}
	for _, join := range s.joins {
		data.Links = append(data.Links, link{
			Source:    join.Tbl1,
			SourceIdx: idxTableNode[join.Tbl1],
			Target:    join.Tbl2,
			TargetIdx: idxTableNode[join.Tbl2],
			Value:     join.Occurrences,
			Type:      "join",
		})
	}

	txTablesMap := make(map[graphKey]int)
	for _, transaction := range s.transactions {
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
		data.Links = append(data.Links, link{
			Source:    key.Tbl1,
			SourceIdx: idxTableNode[key.Tbl1],
			Target:    key.Tbl2,
			TargetIdx: idxTableNode[key.Tbl2],
			Value:     val,
			Type:      "tx",
		})
	}

	return data
}

func createGraphKey(tableA, tableB string) graphKey {
	if tableA < tableB {
		return graphKey{Tbl1: tableA, Tbl2: tableB}
	}
	return graphKey{Tbl1: tableB, Tbl2: tableA}
}

func renderQueryGraph(s *Summary) {
	data := createForceGraphData(s)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Get the assigned port
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		exit("could not create a listener")
	}
	fmt.Printf("Server started at http://localhost:%d\nExit the program with CTRL+C\n", addr.Port)

	// Start the server
	// nolint: gosec,nolintlint // this is all ran locally so no need to care about vulnerabilities around timeouts
	err = http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		serveIndex(w, data)
	}))
	if err != nil {
		exit(err.Error())
	}
}

// Function to dynamically generate and serve index.html
func serveIndex(w http.ResponseWriter, data forceGraphData) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		exit(err.Error())
	}

	tmpl, err := template.New("index").Parse(templateHTML)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	// nolint: gosec,nolintlint // this is all ran locally so no need to care about vulnerabilities around escaping
	if err := tmpl.Execute(w, template.JS(dataBytes)); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		return
	}
}

/*
TODO:
	- New relationship: Add transactions
	- New relationship: FKs
	- Curve links
	- See the join the predicates when hovering the links
	- Different sizes of nodes and links based on table size and relationship occurrences
	- Add a legend
*/

const templateHTML = `<head>
    <style> body { margin: 0; } </style>
    <script src="//unpkg.com/force-graph"></script>
</head>
<body>
    <div id="graph"></div>
    <script>
		let data = {{.}};
        data.links.forEach(link => {
            const a = data.nodes[link.source_idx];
            const b = data.nodes[link.target_idx];
            !a.neighbors && (a.neighbors = []);
            !b.neighbors && (b.neighbors = []);
            a.neighbors.push(b);
            b.neighbors.push(a);

            !a.links && (a.links = []);
            !b.links && (b.links = []);
            a.links.push(link);
            b.links.push(link);
        });

        const highlightNodes = new Set();
        const highlightLinks = new Set();
        let hoverNode = null;

        const Graph = ForceGraph()
        (document.getElementById('graph'))
            .graphData(data)
            .nodeId('id')
            .nodeLabel('id')
            .onLinkHover(link => {
                highlightNodes.clear();
                highlightLinks.clear();

                if (link) {
                    highlightLinks.add(link);
                    highlightNodes.add(link.source);
                    highlightNodes.add(link.target);
                }
            })
            .linkColor(link => {
                if (link.type === 'tx') {
                    return 'rgb(0,255,0)'
                } else if (link.type === 'join') {
                    return 'rgb(255,0,0)'
                } else {
                    return 'rgb(0,0,255)'
                }
            })
            .linkWidth(link => {
                if (highlightLinks.has(link)) {
                    return link.value * 1.5
                }
                return link.value
            })
            .linkLabel('value')
            .autoPauseRedraw(false) // keep redrawing after engine has stopped
            .linkDirectionalParticles(4)
            .linkDirectionalParticleWidth(link => highlightLinks.has(link) ? 4 : 0)
            .nodeCanvasObject((node, ctx, globalScale) => {
                const label = node.id;
                const fontSize = 12/globalScale;
                ctx.font = fontSize+'px Sans-Serif';
                const textWidth = ctx.measureText(label).width;
                const bckgDimensions = [textWidth, fontSize].map(n => n + fontSize * 1); // some padding

                ctx.fillStyle = 'rgb(0,14,71)';
                ctx.fillRect(node.x - bckgDimensions[0] / 2, node.y - bckgDimensions[1] / 2, ...bckgDimensions);

                ctx.textAlign = 'center';
                ctx.textBaseline = 'middle';
                ctx.fillStyle = 'rgb(255,255,255)';
                if (highlightNodes.has(node)) {
                    ctx.fillStyle = node === hoverNode ? 'red' : 'orange';
                }
                ctx.fillText(label, node.x, node.y);

                node.__bckgDimensions = bckgDimensions;

                if (highlightNodes.has(node)) {
                    ctx.beginPath();
                    ctx.fill();
                }
            })
            .onNodeHover(node => {
                highlightNodes.clear();
                highlightLinks.clear();
                if (node) {
                    highlightNodes.add(node);
                    node.neighbors.forEach(neighbor => highlightNodes.add(neighbor));
                    node.links.forEach(link => highlightLinks.add(link));
                }

                hoverNode = node || null;
            })
            .d3Force('link').strength(link => {
                return data.links[link.index].value * 0.2
            });
    </script>
</body>`
