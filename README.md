# VT Utilities

The `vt` binary encapsulates several utility tools for Vitess, providing a comprehensive suite for testing, summarizing,
and query analysis.

## Tools Included

- **`vt test`**: A testing utility using the same test files as
  the [MySQL Test Framework](https://github.com/mysql/mysql-server/tree/8.0/mysql-test). It compares the results of
  identical queries executed on both MySQL and Vitess (vtgate), helping to ensure compatibility.
- **`vt keys`**: A utility that analyzes query logs and provides information about queries, tables, joins, and column
  usage.
- **`vt transactions`**: A tool that analyzes query logs to identify transaction patterns and outputs a JSON report
  detailing these patterns.
- **`vt trace`**: A tool that generates execution traces for queries without comparing against MySQL. It helps analyze
  query behavior and performance in Vitess environments.
- **`vt summarize`**: A tool used to summarize or compare trace logs or key logs for easier human consumption.
- **`vt dbinfo`**: A tool that provides information about the database schema, including row counts, useful column
  attributes and relevant subset of global variables.
- **`vt planalyze`**: A tool that uses `vt keys` output plus a suggested VSchema to analyze potential query plans without
  bringing up a cluster. Queries are classified as:
  - **Perfect**: Single-shard queries.
  - **Good**: Single-route plans, no expensive multi-shard operations.
  - **Valid**: Plans requiring vtgate-level work. Acceptable if rarely used, but potentially slow for frequent queries.

## Installation

You can install `vt` using the following command:

```bash
go install github.com/vitessio/vt/go/vt@latest
```

## Testing Methodology

To verify compatibility and correctness, the testing strategy involves running identical queries on both MySQL and
vtgate, followed by a comparison of results. The process includes:

1. **Query Execution**: Each test query is executed on both MySQL and vtgate.
2. **Result Comparison**: The returned data, result set structure (column types, order), and errors are compared.
3. **Error Handling**: Any errors are checked to ensure vtgate produces the same error types as MySQL.

This dual-testing strategy ensures high confidence in vtgate's compatibility with MySQL.

### Sharded Testing Strategy

Vitess operates in a sharded environment, presenting unique challenges, especially during schema changes (DDL). The
`vt test` tool handles these by converting DDL statements into VSchema commands.

Here's an example of running `vt test`:

```bash
vt test --sharded t/basic.test  # Runs tests on a sharded database
```

Custom schemas and configurations can be applied using directives.
Run `vt test --help`, and check out `directives.test` for more examples.

## Tracing and Query Analysis

Vitess provides two main approaches for tracing query execution:

### Comparative Tracing with `vt test`

`vt test` can generate traces while comparing behavior with MySQL using the `--trace-file` flag:

```bash
vt test --sharded --trace-file=trace-log.json t/tpch.test
```

### Standalone Tracing with `vt trace`

`vt trace` focuses solely on analyzing query execution in Vitess without MySQL comparison:

```bash
# With VSchema and backup initialization
vt trace --vschema=t/vschema.json --backup-path=/path/to/backup --number-of-shards=4 t/tpch.test > trace-log.json
```

`vt trace` accepts most of the same configuration flags as `vt test`, including:

- `--sharded`: Enable auto-sharded mode - uses primary keys as sharding keys. Not a good idea for a production
  environment, but can be used to ensure that all queries work in a sharded environment.
- `--vschema`: Specify the VSchema configuration
- `--backup-path`: Initialize from a backup
- `--number-of-shards`: Specify the number of shards to bring up
- Other database configuration flags

Both `vt trace` and `vt keys` support different input file formats through the `--input-type` flag:

Example using different input types:

```bash
# Analyze SQL file or slow query log
vt trace slow-query.log > trace-log.json

# Analyze MySQL general query log
vt trace --input-type=mysql-log general-query.log > trace-log.json

# Analyze VTGate query log
vt trace --input-type=vtgate-log vtgate-querylog.log > trace-log.json
```

Both types of trace logs can be analyzed using `vt summarize`:

```bash
vt summarize trace-log.json  # Summarize a single trace
vt summarize trace-log1.json trace-log2.json  # Compare two traces
```

## Key Analysis Workflow

`vt keys` analyzes query logs and outputs detailed information about tables, columns usage and joins in queries.
This data can be summarized using `vt summarize`.
Here's a typical workflow:

1. **Run `vt keys` to analyze queries**:

   ```bash
   # Analyze an SQL file or slow query log
   vt keys slow-query.log > keys-log.json

   # Analyze a MySQL general query log
   vt keys --input-type=mysql-log general-query.log > keys-log.json
   
   # Analyze VTGate query log
   vt trace --input-type=vtgate-log vtgate-querylog.log > trace-log.json
   ```

This command generates a `keys-log.json` file that contains a detailed analysis of table and column usage from the
queries.

2. **Summarize the `keys-log` using `vt summarize`**:

   ```bash
   vt summarize keys-log.json
   ```

   This command summarizes the key analysis, providing insight into which tables and columns are used across queries,
   and how frequently they are involved in filters, groupings, and joins.  
   [Here](https://github.com/vitessio/vt/blob/main/go/summarize/testdata/keys-summary.md) is an example summary report.

   If you have access to the running database, you can use `vt dbinfo > dbinfo.json` and pass it to `summarize` so 
   that the analysis can take into the account the additional information from the database schema and configuration:

   ```bash
   vt summarize keys-log.json dbinfo.json
   ```

## Transaction Analysis with vt transactions
The `vt transactions` command is designed to analyze query logs and identify patterns of transactional queries. 
It processes the logs to find sequences of queries that form transactions and outputs a JSON report summarizing these patterns.
Read more about how to use and how to read the output in the [vt transactions documentation](./go/transactions/README.md).

## Using `--backup-path` Flag

The `--backup-path` flag allows `vt test` and `vt trace` to initialize tests from a database backup rather than an empty database.
This is particularly helpful when verifying compatibility during version upgrades or testing stateful operations.

Example:
```bash
vt test --backup-path /path/to/backup -vschema t/vschema.json t/basic.test
```

## Contributing

We welcome contributions in the following areas:

- Writing documentation on how to use the framework
- Triaging issues
- Submitting new test cases
- Fixing bugs in the test framework
- Adding features from the MySQL test framework that are missing in this implementation

After cloning the repo, make sure to run

```bash
make install-hooks
```

to install the pre-commit hooks.

## License

Vitess Tester is licensed under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for more information.

## Acknowledgments

Vitess Tester started as a fork from [pingcap/mysql-tester](https://github.com/pingcap/mysql-tester). We thank the
original authors for their foundational work.