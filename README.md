# VT utils

The `vt` binary encapsulates several utils tools for Vitess.
It includes the following utilities:

- `vt tester`, a testing utility, using the same test files as the [MySQL Test Framework](https://github.com/mysql/mysql-server/tree/8.0/mysql-test).
- `vt benchstat`, a benchmark utility that compares the query planning performance of two Vitess versions.

## Install

Install `vt` using the following command:

```
go install github.com/vitessio/vitess-tester@latest
```

## Testing methodology

To ensure compatibility and correctness, our testing strategy involves running identical queries against both MySQL and vtgate, then comparing the results and errors from the two systems. This approach helps us verify that vtgate behaves as expected in a variety of scenarios, mimicking MySQL's behavior as closely as possible.

The process is straightforward:
* *Query Execution*: We execute each test query against both the MySQL database and vtgate.
* *Result Comparison*: We then compare the results from MySQL and vtgate. This comparison covers not just the data returned but also the structure of the result set, including column types and order.
* *Error Handling*: Equally important is the comparison of errors. When a query results in an error, we ensure that vtgate produces the same error type as MySQL. This step is crucial for maintaining consistency in error handling between the two systems.
* By employing this dual-testing strategy, we aim to achieve a high degree of confidence in vtgate's compatibility with MySQL, ensuring that applications can switch between MySQL and vtgate with minimal friction and no surprises.


## Sharded Testing Strategy
When testing with Vitess, handling sharded databases presents unique challenges, particularly with schema changes (DDL). To navigate this, we intercept DDL statements and convert them into VSchema commands, enabling us to adaptively manage the sharded schema without manual intervention.

A critical aspect of working with Vitess involves defining sharding keys for all tables. Our strategy prioritizes using primary keys as sharding keys wherever available, leveraging their uniqueness and indexing properties. In cases where tables lack primary keys, we resort to using all columns as the sharding key. While this approach may not yield optimal performance due to the broad distribution and potential hotspots, it's invaluable for testing purposes. It ensures that our queries are sharded environment-compatible, providing a robust test bed for identifying issues in sharded setups.

This method allows us to thoroughly test queries within a sharded environment, ensuring that applications running on Vitess can confidently handle both uniform and edge-case scenarios. By prioritizing functional correctness over performance in our testing environment, we can guarantee that Vitess behaves as expected under a variety of sharding configurations.


## How to use

After installing the `vt` binary, you need to have Vitess installed and in your path. 
To run `vt tester` and Vitess, you will need to set the `VTDATAROOT` and `VTROOT` environment variables.
You can do this, and set up the Vitess environment by running the following command:

```sh
source build.env
```

Basic usage:
```
Test the given workload against both Vitess and MySQL.

Usage:
  vt tester  [flags]

Examples:
vt tester 

Flags:
  -h, --help                       help for tester
      --log-level string           The log level of vitess-tester: info, warn, error, debug. (default "error")
      --olap                       Use OLAP to run the queries.
      --sharded                    Run all tests on a sharded keyspace and using auto-vschema. This cannot be used with either -vschema or -vtexplain-vschema.
      --trace-file string          Do a vexplain trace on all queries and store the output in the given file.
      --vschema string             Disable auto-vschema by providing your own vschema file. This cannot be used with either -vtexplain-vschema or -sharded.
      --vtexplain-vschema string   Disable auto-vschema by providing your own vtexplain vschema file. This cannot be used with either -vschema or -sharded.
      --xunit                      Get output in an xml file instead of errors directory
```
It will bring up an entire Vitess cluster on 127.0.0.1, unsharded or sharded depending on the configuration. MySQL and VTGate both start with root and no password configured.

```sh
vt tester t/example.test # run a specified test
vt tester t/example1.test t/example2.test  t/example3.test # separate different tests with one or more spaces
vt tester t/*.test   # wildcards can be used
vt tester https://raw.githubusercontent.com/vitessio/vitess-tester/main/t/basic.test # can also be run against an URL
vt tester --vtexplain-vschema t/vtexplain-vschema.json t/vtexplain.test # run a test with a custom vschema
```

The test files can be amended with directives to control the testing process. Check out `directives.test` to see examples of what directives are available. 

## Tracing and comparing execution plans

`vt tester` can run in tracing mode. When it does, it will not only run the tests but also generate a trace of the query execution plan. 
The trace is created using `vexplain trace`, a tool that provides detailed information about how a query is executed.

To run `vt tester` in tracing mode, use the `--trace` flag:

```bash
vt tester --sharded --trace=trace-log.json t/tpch.test
```

This will create a trace log, which you can then either summarize using `vt benchstat` or compare with another trace log.

Running `vt benchstat` will provide a summary of the trace log, including the number of queries, the number of rows returned, and the time taken to execute the queries.

```bash
vt benchstat trace-log.json
```

To compare two trace logs, use the `vt benchstat` command with two trace logs as arguments:

```bash
vt benchstat trace-log1.json trace-log2.json
```

## Contributing

Contributions are welcomed and greatly appreciated. You can help by:

- writing user document about how to use this framework
- triaging issues
- submitting new test cases
- fixing bugs of this test framework
- adding features that mysql test has but this implementation does not

In case you have any problem, discuss with us here on the repo

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## License

Vitess Tester is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.

## Acknowledgements

Vitess Tester was started as a fork from [pingcap/mysql-tester](https://github.com/pingcap/mysql-tester). We would like to thank the original authors for their work.
