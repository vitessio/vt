# Query Analysis Report

**Date of Analysis**: testdata/keys-log.json  
**Analyzed File**: `2024-01-01 01:02:03`

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
|l_returnflag|WHERE|6%|
|l_shipmode|WHERE RANGE|6%|
|l_shipmode|GROUP|6%|
|l_suppkey|JOIN|39%|
|l_commitdate|WHERE RANGE|28%|
|l_receiptdate|WHERE RANGE|28%|
|l_shipdate|WHERE RANGE|22%|
|l_orderkey|GROUP|17%|
|l_partkey|JOIN|17%|
|l_suppkey|JOIN RANGE|17%|

#### Table: `orders` (11 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|o_orderkey|JOIN|83%|
|o_comment|WHERE RANGE|8%|
|o_orderkey|WHERE RANGE|8%|
|o_orderkey|GROUP|8%|
|o_orderpriority|GROUP|8%|
|o_orderstatus|WHERE|8%|
|o_shippriority|GROUP|8%|
|o_totalprice|GROUP|8%|
|o_custkey|JOIN|58%|
|o_orderdate|WHERE RANGE|42%|
|o_orderdate|GROUP|17%|

#### Table: `nation` (10 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|n_nationkey|JOIN|91%|
|n_name|WHERE|27%|
|n_regionkey|JOIN|27%|
|n_name|GROUP|18%|

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
|c_nationkey|JOIN|50%|
|c_custkey|GROUP|38%|
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
|p_brand|GROUP|17%|
|p_name|WHERE RANGE|17%|
|p_size|WHERE RANGE|17%|
|p_size|GROUP|17%|
|p_type|WHERE|17%|
|p_type|WHERE RANGE|17%|
|p_type|GROUP|17%|

#### Table: `partsupp` (4 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|ps_suppkey|JOIN|60%|
|ps_partkey|JOIN|40%|
|ps_partkey|GROUP|40%|
|ps_suppkey|WHERE RANGE|20%|

#### Table: `region` (2 reads and 1 writes)
|Column|Position|Used %|
|---|---|---|
|r_name|WHERE|67%|
|r_regionkey|JOIN|67%|

## Tables Joined
```
customer ↔ nation
└─ customer.c_nationkey = nation.n_nationkey 100%
```

```
customer ↔ orders
└─ customer.c_custkey = orders.o_custkey 100%
```

```
customer ↔ supplier
└─ customer.c_nationkey = supplier.s_nationkey 100%
```

```
lineitem ↔ lineitem
├─ lineitem.l_orderkey = lineitem.l_orderkey 50%
└─ lineitem.l_suppkey != lineitem.l_suppkey 50%
```

```
lineitem ↔ orders
└─ lineitem.l_orderkey = orders.o_orderkey 100%
```

```
lineitem ↔ part
└─ lineitem.l_partkey = part.p_partkey 100%
```

```
lineitem ↔ partsupp
├─ lineitem.l_partkey = partsupp.ps_partkey 50%
└─ lineitem.l_suppkey = partsupp.ps_suppkey 50%
```

```
lineitem ↔ supplier
└─ lineitem.l_suppkey = supplier.s_suppkey 100%
```

```
nation ↔ region
└─ nation.n_regionkey = region.r_regionkey 100%
```

```
nation ↔ supplier
└─ nation.n_nationkey = supplier.s_nationkey 100%
```

```
part ↔ partsupp
└─ part.p_partkey = partsupp.ps_partkey 100%
```

```
partsupp ↔ supplier
└─ partsupp.ps_suppkey = supplier.s_suppkey 100%
```

## Failures
|Query|Error|
|---|---|
|I am a failing query;|syntax error at position 2 near 'I'|

