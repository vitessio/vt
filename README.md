# Vitess Tester

Vitess tester tests Vitess using the same test files as the [MySQL Test Framework](https://github.com/mysql/mysql-server/tree/8.0/mysql-test).

## Install

```
go install github.com/vitessio/vitess-tester/src/vitess-tester@latest
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

Build the `vitess-tester` binary:
```sh
make
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
        run all tests on a sharded keyspace
  -stderrthreshold value
        logs at or above this threshold go to stderr (default 2)
  -test-dir string
        Directory for the test files (default "./t/")
  -topo-flavor string
        choose a topo server from etcd2, zk2 or consul (default "etcd2")
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
  -vschema string
        Disable auto-vschema by providing your own vschema file
  -xunit
        Get output in an xml file instead of errors directory
```

It will bring up a MySQL server and a vtgate server on 127.0.0.1, both with root and no password configured.

```sh
./vitess-tester # run all the tests
./vitess-tester t/example.test # run a specified test
./vitess-tester t/example1.test t/example2.test  t/example3.test # seperate different tests with one or more spaces
./vitess-tester t/*.test   # wildcards can be used
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
