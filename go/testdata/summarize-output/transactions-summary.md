# Query Analysis Report

**Date of Analysis**: 2024-01-01 01:02:03  
**Analyzed File**: `../testdata/transactions-output/small-slow-query-transactions.json`

## Transaction Patterns

### Pattern 1 (Observed 2 times)


Tables Involved: tblA, tblB
### Query Patterns
1. **UPDATE** on `tblA`  
   Predicates: tblA.foo = 0 AND tblA.id = ?

2. **UPDATE** on `tblB`  
   Predicates: tblB.bar = 0 AND tblB.id = ?

### Shared Predicate Values
* Value 0 applied to:
  - tblA.foo
  - tblB.bar
