# VT Transactions

The vt transactions command is a sub-command of the vt toolset, designed to analyze query logs, identify transaction patterns, and produce a JSON report summarizing these patterns. 
This tool is particularly useful for understanding complex transaction behaviors, optimizing database performance, choosing sharding strategy, and auditing transactional queries.

## Usage

The basic usage of vt transactions is:

```bash
vt transactions querylog.log > report.json
```

 * querylog.log: The input query log file. This can be in various formats, such as SQL files, slow query logs, MySQL general query logs, or VTGate query logs.
 * report.json: The output JSON file containing the transaction patterns.

### Supported Input Types

`vt transactions` supports different input file formats through the --input-type flag:
 * Default: Assumes the input is an SQL file or a slow query log. A SQL script would also fall under this category.
 * MySQL General Query Log: Use --input-type=mysql-log for MySQL general query logs.
 * VTGate Query Log: Use --input-type=vtgate-log for VTGate query logs.

## Understanding the JSON Output

The output JSON file contains an array of transaction patterns, each summarizing a set of queries that commonly occur together within transactions. Here’s a snippet of the JSON output:

```json
{
  "query-signatures": [
    "update pos_reports where id = :0 set `csv`, `error`, intraday, pos_type, ...",
    "update pos_date_requests where cache_key = :1 set cache_value"
  ],
  "predicates": [
    "pos_date_requests.cache_key = ?",
    "pos_reports.id = ?"
  ],
  "count": 223
}
```

### Fields Explanation

 * query-signatures: An array of generalized query patterns involved in the transaction. Placeholders like :0, :1, etc., represent variables in the queries.
 * predicates: An array of predicates (conditions) extracted from the queries, generalized to identify patterns.
 * count: The number of times this transaction pattern was observed in the logs.

### Understanding predicates

The predicates array lists the conditions used in the transactional queries, with variables generalized for pattern recognition.
 * Shared Variables: If the same variable is used across different predicates within a transaction, it is assigned a numerical placeholder (e.g., 0, 1, 2). This indicates that the same variable or value is used in these predicates.
 * Unique Variables: Variables that are unique to a single predicate are represented with a ?.

### Example Explained

Consider the following predicates array:

```json
{
  "predicates": [
    "timesheets.day = ?",
    "timesheets.craft_id = ?",
    "timesheets.store_id = ?",
    "dailies.day = 0",
    "dailies.craft_id = 1",
    "dailies.store_id = 2",
    "tickets.day = 0",
    "tickets.craft_id = 1",
    "tickets.store_id = 2"
  ]
}
```

 * Shared Values: Predicates with the same value across different conditions are assigned numerical placeholders (0, 1, 2), indicating that the same variable or value is used in these predicates.
 * For example, `dailies.craft_id = 1` and `tickets.craft_id = 1` share the same variable or value (represented as 1).
 * Unique Values: Predicates used only once are represented with ?, indicating a unique or less significant variable in the pattern.
 * For example, `timesheets.day = ?` represents a unique value for day.

This numbering helps identify the relationships between different predicates in the transaction patterns and can be used to optimize queries or understand transaction scopes.

## Practical Use Cases

 * Optimization: Identify frequently occurring transactions to optimize database performance.
 * Sharding Strategy: When implementing horizontal sharding, it’s crucial to ensure that as many transactions as possible are confined to a single shard. The insights from vt transactions can help in choosing appropriate sharding keys for your tables to achieve this.
 * Audit: Analyze transactional patterns for security audits or compliance checks.
 * Debugging: Understand complex transaction behaviors during development or troubleshooting.
