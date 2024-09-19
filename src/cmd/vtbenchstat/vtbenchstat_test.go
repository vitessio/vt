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

package main

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var tf1 = TraceFile{
	Name: "test",
	Queries: []TracedQuery{{
		Query:      "select * from music",
		LineNumber: "1",
		Trace: Trace{
			OperatorType:       "Route",
			Variant:            "Scatter",
			NoOfCalls:          1,
			AvgNumberOfRows:    16,
			MedianNumberOfRows: 16,
			ShardsQueried:      8,
		},
	}, {
		Query:      "select tbl.foo, tbl2.bar from tbl join tbl2 on tbl.id = tbl2.id order by tbl.baz",
		LineNumber: "2",
		Trace: Trace{
			OperatorType:       "Sort",
			Variant:            "Memory",
			NoOfCalls:          1,
			AvgNumberOfRows:    16,
			MedianNumberOfRows: 16,
			Inputs: []Trace{{
				OperatorType:       "Join",
				Variant:            "Apply",
				NoOfCalls:          1,
				AvgNumberOfRows:    16,
				MedianNumberOfRows: 16,
				Inputs: []Trace{{
					OperatorType:       "Route",
					Variant:            "Scatter",
					NoOfCalls:          1,
					AvgNumberOfRows:    10,
					MedianNumberOfRows: 10,
					ShardsQueried:      8,
				}, {
					OperatorType:       "Route",
					Variant:            "EqualUnique",
					NoOfCalls:          10,
					AvgNumberOfRows:    1,
					MedianNumberOfRows: 1,
					ShardsQueried:      10,
				}},
			}},
		},
	}},
}
var tf2 = TraceFile{
	Name: "test",
	Queries: []TracedQuery{{
		Query:      "select * from music",
		LineNumber: "1",
		Trace: Trace{
			OperatorType:       "Route",
			Variant:            "Scatter",
			NoOfCalls:          1,
			AvgNumberOfRows:    16,
			MedianNumberOfRows: 16,
			ShardsQueried:      7,
		},
	}, {
		Query:      "select tbl.foo, tbl2.bar from tbl join tbl2 on tbl.id = tbl2.id order by tbl.baz",
		LineNumber: "2",
		Trace: Trace{
			OperatorType:       "Route",
			Variant:            "Scatter",
			NoOfCalls:          1,
			AvgNumberOfRows:    16,
			MedianNumberOfRows: 16,
			ShardsQueried:      8,
		},
	}},
}

func TestSummary(t *testing.T) {
	t.Run("tf1", func(t *testing.T) {
		sb := &strings.Builder{}
		printSummary(sb, 80, noHighlight, tf1)
		assert.Equal(t, `Query: select * from music
Line # 1
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |        16 |              0 |              8 |
+-------------+-----------+----------------+----------------+

Query: select tbl.foo, tbl2.bar from tbl join tbl2 on tbl.id = tbl2.id order ...
Line # 2
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|          11 |        20 |             16 |             18 |
+-------------+-----------+----------------+----------------+
`, sb.String())
	})

	t.Run("tf2", func(t *testing.T) {
		sb := &strings.Builder{}
		printSummary(sb, 80, noHighlight, tf2)
		assert.Equal(t, `Query: select * from music
Line # 1
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |        16 |              0 |              7 |
+-------------+-----------+----------------+----------------+

Query: select tbl.foo, tbl2.bar from tbl join tbl2 on tbl.id = tbl2.id order ...
Line # 2
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |        16 |              0 |              8 |
+-------------+-----------+----------------+----------------+
`, sb.String())
	})
}

func TestCompareFiles(t *testing.T) {
	sb := &strings.Builder{}
	compareTraces(sb, 80, noHighlight, tf1, tf2)
	s := sb.String()
	want := `Query: select * from music
Line # 1 (significant)
+----------------+------+------+------+----------+
|     Metric     | test | test | Diff | % Change |
+----------------+------+------+------+----------+
| Route Calls    |    1 |    1 |    0 | 0.00%    |
| Rows Sent      |   16 |   16 |    0 | 0.00%    |
| Rows In Memory |    0 |    0 |    0 | NaN%     |
| Shards Queried |    8 |    7 |   -1 | -12.50%  |
+----------------+------+------+------+----------+

Query: select tbl.foo, tbl2.bar from tbl join tbl2 on tbl.id = tbl2.id order ...
Line # 2 (significant)
+----------------+------+------+------+----------+
|     Metric     | test | test | Diff | % Change |
+----------------+------+------+------+----------+
| Route Calls    |   11 |    1 |  -10 | -90.91%  |
| Rows Sent      |   20 |   16 |   -4 | -20.00%  |
| Rows In Memory |   16 |    0 |  -16 | -100.00% |
| Shards Queried |   18 |    8 |  -10 | -55.56%  |
+----------------+------+------+------+----------+

Summary:
- 2 out of 2 queries showed significant change
- Average change in Route Calls: -83.33%
- Average change in Data Sent: -11.11%
- Average change in Rows In Memory: -100.00%
- Average change in Shards Queried: -42.31%
`
	if s != want {
		panic("oh noes")
	}
	assert.Equal(t, want, s)
}
