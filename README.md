# Vitess Tester

Vitess tester tests Vitess using the same test files as the [MySQL Test Framework](https://github.com/mysql/mysql-server/tree/8.0/mysql-test).

## Install

If you only want the `vitess-tester` binary, install it using the following command:

```
go install github.com/vitessio/vitess-tester@latest
```

The `vtbenchstat` binary can be had by running:

```
go install github.com/vitessio/vitess-tester/src/cmd/vtbenchstat@latest
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

After installing the `vitess-tester` binary, you need to have Vitess installed and in your path. 
To run `vitess-tester` and Vitess, you will need to set the `VTDATAROOT` and `VTROOT` environment variables.
You can do this, and set up the Vitess environment by running the following command:

```sh
source build.env
```

Basic usage:
```
Usage of ./vitess-tester:
  -alsologtostderr
        log to standard error as well as files
  -force-base-tablet-uid int
        force assigning tablet ports based on this seed
  -force-port-start int
        force assigning ports based on this seed
  -force-vtdataroot string
        force path for VTDATAROOT, which may already be populated
  -is-coverage
        whether coverage is required
  -keep-data
        don't delete the per-test VTDATAROOT subfolders (default true)
  -log-level string
        The log level of vitess-tester: info, warn, error, debug. (default "error")
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -log_link string
        If non-empty, add symbolic links in this directory to the log files
  -logbuflevel int
        Buffer log messages logged at this level or lower (-1 means don't buffer; 0 means buffer INFO only; ...). Has limited applicability on non-prod platforms.
  -logtostderr
        log to standard error instead of files
  -olap
        Use OLAP to run the queries.
  -partial-keyspace
        add a second keyspace for sharded tests and mark first shard as moved to this keyspace in the shard routing rules
  -perf-test
        include end-to-end performance tests
  -sharded
        Run all tests on a sharded keyspace and using auto-vschema. This cannot be used with either -vschema or -vtexplain-vschema.
  -stderrthreshold value
        logs at or above this threshold go to stderr (default 2)
  -topo-flavor string
        choose a topo server from etcd2, zk2 or consul (default "etcd2")
  -trace string
        Do a vexplain trace on all queries and store the output in the given file.
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
  -vschema string
        Disable auto-vschema by providing your own vschema file. This cannot be used with either -vtexplain-vschema or -sharded.
  -vtexplain-vschema string
        Disable auto-vschema by providing your own vtexplain vschema file. This cannot be used with either -vschema or -sharded.
  -xunit
        Get output in an xml file instead of errors directory
```

It will bring up an entire Vitess cluster on 127.0.0.1, unsharded or sharded depending on the configuration. MySQL and VTGate both start with root and no password configured.

```sh
vitess-tester t/example.test # run a specified test
vitess-tester t/example1.test t/example2.test  t/example3.test # separate different tests with one or more spaces
vitess-tester t/*.test   # wildcards can be used
vitess-tester https://raw.githubusercontent.com/vitessio/vitess-tester/main/t/basic.test # can also be run against an URL
vitess-tester -vtexplain-vschema t/vtexplain-vschema.json t/vtexplain.test # run a test with a custom vschema
```

The test files can be amended with directives to control the testing process. Check out `directives.test` to see examples of what directives are available. 

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
