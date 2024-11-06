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
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func tf1() readingSummary {
	return readingSummary{
		Name: "test",
		TracedQueries: []TracedQuery{{
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
}

func tf2() readingSummary {
	return readingSummary{
		Name: "test",
		TracedQueries: []TracedQuery{{
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
}

func TestSummary(t *testing.T) {
	t.Run("tf1", func(t *testing.T) {
		sb := &strings.Builder{}
		printTraceSummary(sb, 80, noHighlight, tf1())
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
		printTraceSummary(sb, 80, noHighlight, tf2())
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
	compareTraces(sb, 80, noHighlight, tf1(), tf2())
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
	assert.Equal(t, want, s)
}

func TestSummarizeTraceFile(t *testing.T) {
	file := readTraceFile("testdata/trace-log.json")
	sb := &strings.Builder{}
	printTraceSummary(sb, 80, noHighlight, file)
	expected := `Query: INSERT INTO region (R_REGIONKEY, R_NAME, R_COMMENT) VALUES (1, 'ASIA',...
Line # 80
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              1 |
+-------------+-----------+----------------+----------------+

Query: INSERT INTO nation (N_NATIONKEY, N_NAME, N_REGIONKEY, N_COMMENT) VALUE...
Line # 84
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              1 |
+-------------+-----------+----------------+----------------+

Query: INSERT INTO supplier (S_SUPPKEY, S_NAME, S_ADDRESS, S_NATIONKEY, S_PHO...
Line # 90
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              1 |
+-------------+-----------+----------------+----------------+

Query: INSERT INTO part (P_PARTKEY, P_NAME, P_MFGR, P_BRAND, P_TYPE, P_SIZE, ...
Line # 96
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              1 |
+-------------+-----------+----------------+----------------+

Query: INSERT INTO partsupp (PS_PARTKEY, PS_SUPPKEY, PS_AVAILQTY, PS_SUPPLYCO...
Line # 100
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              1 |
+-------------+-----------+----------------+----------------+

Query: INSERT INTO customer (C_CUSTKEY, C_NAME, C_ADDRESS, C_NATIONKEY, C_PHO...
Line # 105
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              1 |
+-------------+-----------+----------------+----------------+

Query: INSERT INTO orders (O_ORDERKEY, O_CUSTKEY, O_ORDERSTATUS, O_TOTALPRICE...
Line # 111
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              2 |
+-------------+-----------+----------------+----------------+

Query: INSERT INTO lineitem (L_ORDERKEY, L_PARTKEY, L_SUPPKEY, L_LINENUMBER, ...
Line # 117
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           0 |         0 |              0 |              2 |
+-------------+-----------+----------------+----------------+

Query: select l_returnflag, l_linestatus, sum(l_quantity) as sum_qty, sum(l_e...
Line # 131
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |         8 |              0 |              2 |
+-------------+-----------+----------------+----------------+

Query: select l_orderkey, sum(l_extendedprice * (1 - l_discount)) as revenue,...
Line # 201
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           3 |         4 |              2 |              4 |
+-------------+-----------+----------------+----------------+

Query: select o_orderpriority, count(*) as order_count from orders where o_or...
Line # 226
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |         2 |              0 |              2 |
+-------------+-----------+----------------+----------------+

Query: select n_name, sum(l_extendedprice * (1 - l_discount)) as revenue from...
Line # 249
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           9 |         8 |              0 |             10 |
+-------------+-----------+----------------+----------------+

Query: select sum(l_extendedprice * l_discount) as revenue from lineitem wher...
Line # 275
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |         2 |              0 |              2 |
+-------------+-----------+----------------+----------------+

Query: select supp_nation, cust_nation, l_year, sum(volume) as revenue from (...
Line # 286
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           5 |         4 |              0 |              9 |
+-------------+-----------+----------------+----------------+

Query: select o_year, sum(case when nation = 'INDIA' then volume else 0 end) ...
Line # 327
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           5 |         4 |              0 |             11 |
+-------------+-----------+----------------+----------------+

Query: select nation, o_year, sum(amount) as sum_profit from ( select n_name ...
Line # 366
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           7 |         6 |              0 |             11 |
+-------------+-----------+----------------+----------------+

Query: select c_custkey, c_name, sum(l_extendedprice * (1 - l_discount)) as r...
Line # 400
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |         0 |              0 |              4 |
+-------------+-----------+----------------+----------------+

Query: select ps_partkey, sum(ps_supplycost * ps_availqty) as value from part...
Line # 434
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|          12 |        10 |              0 |             14 |
+-------------+-----------+----------------+----------------+

Query: select l_shipmode, sum(case when o_orderpriority = '1-URGENT' or o_ord...
Line # 463
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |         0 |              0 |              2 |
+-------------+-----------+----------------+----------------+

Query: select c_count, count(*) as custdist from ( select c_custkey, count(o_...
Line # 493
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           5 |         8 |              5 |             10 |
+-------------+-----------+----------------+----------------+

Query: select 100.00 * sum(case when p_type like 'PROMO%' then l_extendedpric...
Line # 515
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |         0 |              0 |              3 |
+-------------+-----------+----------------+----------------+

Query: select p_brand, p_type, p_size, count(distinct ps_suppkey) as supplier...
Line # 530
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           2 |         0 |              0 |              4 |
+-------------+-----------+----------------+----------------+

Query: select c_name, c_custkey, o_orderkey, o_orderdate, o_totalprice, sum(l...
Line # 582
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           2 |         9 |              0 |              4 |
+-------------+-----------+----------------+----------------+

Query: select sum(l_extendedprice* (1 - l_discount)) as revenue from lineitem...
Line # 617
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|          12 |        11 |              0 |             24 |
+-------------+-----------+----------------+----------------+

Query: select s_name, count(*) as numwait from supplier, lineitem l1, orders,...
Line # 695
+-------------+-----------+----------------+----------------+
| Route Calls | Rows Sent | Rows In Memory | Shards Queried |
+-------------+-----------+----------------+----------------+
|           1 |         0 |              0 |              4 |
+-------------+-----------+----------------+----------------+
`
	assert.Equal(t, expected, sb.String())
}
