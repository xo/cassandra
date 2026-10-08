# D30. The driver follows the drivers of dbimp

Status: Decided. Amends D3 and D26.

Ken decided on 2026-10-08 that this repository moves the Cassandra driver
there and drops the name CQL, so that it is consistent with the drivers of
`github.com/xo/dbimp`. The driver follows the same layout, the same rules and
the same tests, with the changes that a native protocol needs.

- The module depends on the standard library, gocql, `github.com/xo/dbimp` and
  `github.com/cockroachdb/apd/v3` for decimals (dbimp D33). Ken approved the
  import of `dbimp` and of `dbimptest` on 2026-10-08, to share the common
  implementations of the options, the DSN, the types and the tests, and so to
  reduce the code that this repository keeps. The dependency runs from this
  repository to `dbimp`, and `dbimp` never imports this one (dbimp D29).
- The driver no longer imports `gopkg.in/inf.v0`. gocql still requires it, so it
  stays in `go.sum` as an indirect module (D26 held that it stays as a direct
  one).
- The files keep the roles of `couchbase/` and `clickhouse/` in `dbimp`:
  `driver.go` registers from `init`, and the other files are the connector, the
  connection, the rows, the types, the options, the errors and the DSN.
- The package holds one document for the product, `docs/CASSANDRA.md`, and the
  type table and the interface table in it come from the code, through
  `dbimptest.TypeTable` and `dbimptest.InterfaceTable`.
- The contract of `dbimptest.RunContract` serves the response of a fake HTTP
  server, so it does not fit a driver of the native protocol. `contract_test.go`
  holds the same checks against the fake session of this repository (open
  question 16).
- The steps of the driver guide of `dbimp` that measure an HTTP interface and
  record its exchanges do not apply (steps 2 to 8, and step 14 with
  recordings).
