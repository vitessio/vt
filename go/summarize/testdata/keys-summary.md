# Query Analysis Report

**Date of Analysis**: 2024-01-01 01:02:03  
**Analyzed File**: `testdata/keys-log.json`

## Tables
|Table Name|Reads|Writes|
|---|---|---|
|lineitem|17|1|
|orders|11|1|
|nation|10|1|
|supplier|8|1|
|customer|7|1|
|part|5|1|
|partsupp|4|1|
|region|2|1|

### Column Usage
#### Table: `lineitem` (17 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|l_orderkey|JOIN|72%|
||GROUP|17%|
|l_suppkey|JOIN|39%|
||JOIN RANGE|17%|
|l_shipdate|WHERE RANGE|33%|
|l_commitdate|WHERE RANGE|28%|
|l_receiptdate|WHERE RANGE|28%|
|l_partkey|JOIN|17%|
|l_discount|WHERE RANGE|6%|
|l_linestatus|GROUP|6%|
|l_quantity|WHERE RANGE|6%|
|l_returnflag|WHERE|6%|
||GROUP|6%|
|l_shipmode|WHERE RANGE|6%|
||GROUP|6%|

#### Table: `orders` (11 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|o_orderkey|JOIN|83%|
||WHERE RANGE|8%|
||GROUP|8%|
|o_custkey|JOIN|58%|
|o_orderdate|WHERE RANGE|42%|
||GROUP|17%|
|o_comment|WHERE RANGE|8%|
|o_orderpriority|GROUP|8%|
|o_orderstatus|WHERE|8%|
|o_shippriority|GROUP|8%|
|o_totalprice|GROUP|8%|

#### Table: `nation` (10 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|n_nationkey|JOIN|91%|
|n_name|WHERE|27%|
||GROUP|18%|
|n_regionkey|JOIN|27%|

#### Table: `supplier` (8 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|s_nationkey|JOIN|78%|
|s_suppkey|JOIN|78%|
|s_comment|WHERE RANGE|11%|
|s_name|GROUP|11%|

#### Table: `customer` (7 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|c_custkey|JOIN|88%|
||GROUP|38%|
|c_nationkey|JOIN|50%|
|c_name|GROUP|25%|
|c_acctbal|GROUP|12%|
|c_address|GROUP|12%|
|c_comment|GROUP|12%|
|c_mktsegment|WHERE|12%|
|c_phone|GROUP|12%|

#### Table: `part` (5 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|p_partkey|JOIN|67%|
|p_brand|WHERE RANGE|17%|
||GROUP|17%|
|p_name|WHERE RANGE|17%|
|p_size|WHERE RANGE|17%|
||GROUP|17%|
|p_type|WHERE|17%|
||WHERE RANGE|17%|
||GROUP|17%|

#### Table: `partsupp` (4 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|ps_suppkey|JOIN|60%|
||WHERE RANGE|20%|
|ps_partkey|JOIN|40%|
||GROUP|40%|

#### Table: `region` (2 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|r_name|WHERE|67%|
|r_regionkey|JOIN|67%|

## Tables Joined
```
lineitem ↔ orders (Occurrences: 10)
└─ lineitem.l_orderkey = orders.o_orderkey

customer ↔ orders (Occurrences: 7)
└─ customer.c_custkey = orders.o_custkey

nation ↔ supplier (Occurrences: 6)
└─ nation.n_nationkey = supplier.s_nationkey

lineitem ↔ supplier (Occurrences: 5)
└─ lineitem.l_suppkey = supplier.s_suppkey

lineitem ↔ part (Occurrences: 3)
└─ lineitem.l_partkey = part.p_partkey

customer ↔ nation (Occurrences: 3)
└─ customer.c_nationkey = nation.n_nationkey

lineitem ↔ lineitem (Occurrences: 2)
├─ lineitem.l_orderkey = lineitem.l_orderkey
└─ lineitem.l_suppkey != lineitem.l_suppkey

lineitem ↔ partsupp (Occurrences: 2)
├─ lineitem.l_partkey = partsupp.ps_partkey
└─ lineitem.l_suppkey = partsupp.ps_suppkey

nation ↔ region (Occurrences: 2)
└─ nation.n_regionkey = region.r_regionkey

partsupp ↔ supplier (Occurrences: 1)
└─ partsupp.ps_suppkey = supplier.s_suppkey

part ↔ partsupp (Occurrences: 1)
└─ part.p_partkey = partsupp.ps_partkey

customer ↔ supplier (Occurrences: 1)
└─ customer.c_nationkey = supplier.s_nationkey

```
## Failures
|Query|Error|Count|
|---|---|---|
|I am a failing query;|syntax error at position 2 near 'I'|2|

