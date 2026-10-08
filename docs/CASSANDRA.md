# Cassandra

This file holds what is known about Apache Cassandra and ScyllaDB for the
driver `cassandra`. The headings are the template of the driver guide of
`dbimp`, [DRIVER.md](https://github.com/xo/dbimp/blob/main/docs/DRIVER.md). The
steps that measure an HTTP interface and record its exchanges do not apply,
because the driver speaks the native protocol through gocql and records no
exchange (D30).

Each fact says how it is known. A fact marked "read" was read in the source of
gocql `v2.1.2`, or of `dbmeta` or `dburl`, on the date of this file. A fact
marked "measured" names the release and the test that measured it. A fact
marked "not measured" names its source.

## Summary

- Apache Cassandra and ScyllaDB are wide-column databases that speak CQL, a
  language like SQL, over the native binary protocol. The driver wraps gocql
  (D6) and registers the one name `cassandra` (D27).
- `dbrun` starts `cassandra-3.11`, `cassandra-4.0`, `cassandra-4.1`,
  `cassandra-5.0` and the `scylla` releases. The release list lives in
  `dbmeta`, and the workflow reads it (D21).
- The scheme in `dburl` is `cql`, with the aliases `ca`, `cassandra`,
  `datastax`, `scy` and `scylla`, and its generator writes the older list of
  hosts (read: `dburl/scheme.go` and `dburl/dsn.go` on 2026-10-08). That
  disagrees with this driver, which reads `cassandra://` only (D31), so `dburl`
  must change, and Ken decided how (D35, open question 19). `usql` imports this driver in
  `drivers/cassandra/cassandra.go`.
- The server reports its version with `SELECT release_version FROM
  system.local` (read: `dbmeta`).

## Requests

The driver sends no HTTP request. gocql opens connections to every node of the
cluster and sends CQL frames over them. The connector creates one session, and
every connection of the `sql.DB` shares it (D17). Authentication is
`gocql.PasswordAuthenticator`, from the user information of the DSN. TLS is on
when a TLS key is in the query.

## The DSN

The DSN is a URL with the one scheme `cassandra`, parsed with `net/url` (D31).
No other scheme, alias or older form is read. A DSN with another scheme is an
error that wraps `dbimp.ErrScheme`.

```text
cassandra://user:password@host1:9042/keyspace?consistency=localQuorum&host=host2
```

The user information is the credentials, the path is the keyspace, and the
host part holds one host. Each `host` key adds one more (D34). Every query key has the default of
`gocql.NewCluster`. A key that the table does not name is an error that wraps
`dbimp.ErrUnknownKey`, and a key that appears twice wraps `dbimp.ErrRepeatedKey`.

| Key | Meaning | Default |
| --- | --- | --- |
| `consistency` | `any`, `one`, `two`, `three`, `quorum`, `all`, `localQuorum`, `eachQuorum` or `localOne` | `quorum` |
| `keyspace` | the keyspace, when the path is empty | none |
| `timeout` | the timeout of one request | 11s |
| `connectTimeout` | the timeout of a new connection | 11s |
| `numConns` | the connections to each host | 2 |
| `ignorePeerAddr` | `true` or `false` | `false` |
| `disableInitialHostLookup` | `true` or `false` | `false` |
| `writeCoalesceWaitTime` | a duration | 200µs |
| `username`, `password` | the credentials, when the user information holds none | none |
| `enableHostVerification`, `certPath`, `keyPath`, `caPath` | TLS | off |
| `host` | one more host | none |

## Types

The table below is the type table of step 10. A test writes it from the code
with `dbimptest.TypeTable`, and fails when it is stale. Each type has one Go
type in `*any`, in `Rows.Next` and in `ColumnTypeScanType`, for a column that
can be NULL too (dbimp D135). A caller scans a nullable value into
`sql.Null[T]`.

<!-- dbimp:types -->
| Wire type | Kind | Go type | Scan type | Database type | Can be NULL |
| --- | --- | --- | --- | --- | --- |
| ascii | string | `string` | `string` | `ASCII` | yes |
| text | string | `string` | `string` | `TEXT` | yes |
| varchar | string | `string` | `string` | `VARCHAR` | yes |
| boolean | boolean | `bool` | `bool` | `BOOLEAN` | yes |
| tinyint | integer | `int64` | `int64` | `TINYINT` | yes |
| smallint | integer | `int64` | `int64` | `SMALLINT` | yes |
| int | integer | `int64` | `int64` | `INT` | yes |
| bigint | integer | `int64` | `int64` | `BIGINT` | yes |
| counter | integer | `int64` | `int64` | `COUNTER` | yes |
| float | float | `float64` | `float64` | `FLOAT` | yes |
| double | float | `float64` | `float64` | `DOUBLE` | yes |
| decimal | decimal | `*apd.Decimal` | `*apd.Decimal` | `DECIMAL` | yes |
| varint | big integer | `*big.Int` | `*big.Int` | `VARINT` | yes |
| blob | binary | `[]byte` | `[]uint8` | `BLOB` | yes |
| uuid | uuid | `uuid.UUID` | `uuid.UUID` | `UUID` | yes |
| timeuuid | uuid | `uuid.UUID` | `uuid.UUID` | `TIMEUUID` | yes |
| inet | ip address | `netip.Addr` | `netip.Addr` | `INET` | yes |
| timestamp | timestamp | `time.Time` | `time.Time` | `TIMESTAMP` | yes |
| date | date | `dbimp.Date` | `dbimp.Date` | `DATE` | yes |
| time | time of day | `dbimp.LocalTime` | `dbimp.LocalTime` | `TIME` | yes |
| duration | interval | `dbimp.Interval` | `dbimp.Interval` | `DURATION` | yes |
| list | array | `[]any` | `[]interface {}` | `LIST<INT>` | yes |
| set | set | `[]any` | `[]interface {}` | `SET<TEXT>` | yes |
| map with a text key | map | `map[string]any` | `map[string]interface {}` | `MAP<TEXT, INT>` | yes |
| map with another key | map | `the map that gocql chooses` | `map[int]string` | `MAP<INT, TEXT>` | yes |
| tuple | tuple | `[]any` | `[]interface {}` | `TUPLE<INT, TEXT>` | yes |
| user defined type | map | `map[string]any` | `map[string]interface {}` | `ADDRESS` | yes |
| vector of float | vector | `dbimp.Vector[float32]` | `dbimp.Vector[float32]` | `VECTOR<FLOAT, 3>` | yes |
| vector of double | vector | `dbimp.Vector[float64]` | `dbimp.Vector[float64]` | `VECTOR<DOUBLE, 3>` | yes |
| vector of int | vector | `dbimp.Vector[int32]` | `dbimp.Vector[int32]` | `VECTOR<INT, 3>` | yes |
| vector of bigint | vector | `dbimp.Vector[int64]` | `dbimp.Vector[int64]` | `VECTOR<BIGINT, 3>` | yes |
| vector of other | array | `[]any` | `[]interface {}` | `VECTOR<TEXT, 3>` | yes |
<!-- /dbimp:types -->

NULL. A NULL column is nil. gocql reports a NULL tuple as a tuple whose elements
are all NULL, so the two cannot be told apart (read). Cassandra stores an empty
collection as NULL, so a scan of one into a slice or a map gives nil and no
error (D10).

A decimal. gocql decodes a decimal into an `*inf.Dec`. The driver does not
import that package. It reads the number as text into an `*apd.Decimal`, and it
writes the wire form itself, the scale as an int32 and the unscaled value as a
two's complement integer (read: `marshal.go` of gocql).

A time. A `date` is a `dbimp.Date`, a `time` is a `dbimp.LocalTime`, and a
`duration` is a `dbimp.Interval`, which holds months, days and nanoseconds as
CQL does. A `timestamp` is a `time.Time`.

An address. gocql writes an IPv4 address in 16 bytes and does not pass on the
length, so the driver reports an IPv4 address as an IPv4 `netip.Addr`.

A collection. A `list`, a `set` and a `tuple` are a `[]any` of the Go types of
their elements. A `map` with a text key is a `map[string]any`. A user defined
type is a `map[string]any` with the fields of the type. A map with any other
key stays the map that gocql chooses, such as `map[int]string`, because no kind
of the types document of `dbimp` holds it (D35). A vector of
numbers is a `dbimp.Vector[T]`.

## Parameters

Parameters are positional, at `?` markers. A named argument from `sql.Named` is
an error that wraps `dbimp.ErrNotSupported`, because gocql has no call that
binds by name. The driver binds the canonical types of the table as arguments
too: a `dbimp.Date`, a `dbimp.LocalTime`, a `dbimp.Interval`, an
`*apd.Decimal`, a `netip.Addr`, a `uuid.UUID` and a `dbimp.Vector`. gocql
`v2.1.2` cannot bind a tuple, and a tuple is written as a CQL literal (D17).

## Transactions

CQL has no transactions, and `BeginTx` fails with an error that wraps
`dbimp.ErrNotSupported` (dbimp D20). A batch is a statement of its own, and a
lightweight transaction is a `IF` clause.

## Errors

Every error that gocql or the server reports reaches the caller wrapped with
`%w` (D9), so `errors.As` finds `*gocql.RequestErrSyntax` and the other types of
gocql. A result that fails after a row reached the caller wraps
`dbimp.ErrIncomplete` (dbimp D21). The driver never returns `driver.ErrBadConn`
from a statement, because every connection shares one session and a second try
meets the same fault. It returns it from `ResetSession` after the connector
closes, which is the one case where the connection is broken and no statement
reached the server.

## Cancellation and timeouts

The context of a call stops the request that runs and each later page fetch.
`WithTimeout` ends a statement after a time, and it covers every page of its
rows. The `timeout` key of the DSN is the timeout of one request in gocql.

## Statements

A statement is one CQL statement at a time. The driver does not split a string
that holds several. A `USE` statement is refused by gocql, so the keyspace is
the one of the DSN, or the one that `WithDatabase` names, which needs protocol
version 5.

## Principals

`dbrun` names one principal for each Cassandra and ScyllaDB release, the
administrator `cassandra` (measured: `dbrun dsn --json cassandra-5.0` on
2026-10-08). So the integration tests run as that user only, and open question 18
in [PLAN.md](PLAN.md) holds the request for an ordinary one. The ordinary user is not
measured: what a role without `SELECT` on `system.local` gets from the ping and
from the version statement.

## Flavors

ScyllaDB speaks the same protocol and the same CQL. It differs from Cassandra in
a few places, such as `SimpleStrategy` for a keyspace, which a ScyllaDB with
tablets refuses (read: `integration_test.go`). The tests run on both.

## Interfaces

The table below is the interface table of step 10. A test writes it with
`dbimptest.InterfaceTable`, and fails when it is stale.

<!-- dbimp:interfaces -->
| Interface | Implemented | Reason |
| --- | --- | --- |
| `driver.DriverContext` | yes | OpenConnector parses the DSN once, for every connection. |
| `driver.Connector` | yes | The connector owns the one gocql session, which every connection shares (D17). |
| `io.Closer on the connector` | yes | Close closes the session. |
| `driver.Pinger` | yes | Ping reads the release version of the node that answers. |
| `driver.SessionResetter` | yes | It returns driver.ErrBadConn after the connector is closed, so that database/sql discards the connection. A connection holds no other state. |
| `driver.Validator` | yes | It reports whether the connector is still open. |
| `driver.NamedValueChecker` | yes | It keeps an Option, and the values that the driver binds with a type of its own: a decimal, the types of dbimp, an address and a UUID (D25). |
| `driver.QueryerContext` | yes | gocql binds each argument at its ? marker. |
| `driver.ExecerContext` | yes | RowsAffected and LastInsertId fail, because Cassandra reports neither. |
| `driver.ConnPrepareContext` | yes | A prepared statement holds the text, and gocql prepares and caches it when it runs. |
| `driver.ConnBeginTx` | yes | BeginTx fails with dbimp.ErrNotSupported, because CQL has no transactions (D9). |
| `driver.RowsColumnScanner` | yes | gocql decodes straight into the destination of the caller, and a canonical value reaches any, a Scanner and the types of dbimp. |
| `driver.RowsNextResultSet` | no | A statement has one result. |
| `driver.RowsColumnTypeScanType` | yes | From the type of the column, with one Go type for each type. |
| `driver.RowsColumnTypeDatabaseTypeName` | yes | The CQL type in upper case, such as LIST<INT>. |
| `driver.RowsColumnTypeLength` | no | No CQL type has a length. |
| `driver.RowsColumnTypeNullable` | yes | It reports that the nullability is not known, because gocql does not say which column is a primary key. |
| `driver.RowsColumnTypePrecisionScale` | no | A decimal column has no precision or scale in its type. |
<!-- /dbimp:interfaces -->

## Faults

The faults of the driver that `usql` used before, which this driver must not
repeat (read: `docs/BACKLOG.md`): a NULL read as a zero value, an error that
became `driver.ErrBadConn` and ran a write twice, a session for each
connection, and a logger that the package set globally.

## Second opinions

Gemini 3.1 Pro and DeepSeek V4 reviewed the design on 2026-09-27, and
[DESIGN.md](DESIGN.md) records where they agreed and where they differed. They
did not review the move to the types of `dbimp`.

## Open questions

The open questions that wait for Ken are at the end of [PLAN.md](PLAN.md).

## Integration tests

The integration tests read `CASSANDRA_DSN`, create a keyspace of their own, and
drop it. They run as the administrator, the one principal that `dbrun` names. On
2026-10-08, `dbrun` started each release and the tests passed on all of them
with `go test -race -count=1 -run Integration ./...`:

| Release | Server | Protocol |
| --- | --- | --- |
| `cassandra-3.11` | Cassandra 3.11.19 | v4 |
| `cassandra-5.0` | Cassandra 5.0.9 | v5 |
| `scylla-2025.1` | ScyllaDB 2025.1.16 | v4 |
| `scylla-2026.3` | ScyllaDB 2026.3.3 | v4 |

The round trip of step 14a, `TestIntegrationRoundTrip`, stores values of each type
in a column with `dbimptest.RoundTrip`, as an argument and as a literal, and reads
them through `Rows.Scan`. A test skips a difference between releases with the
reason: `cassandra-3.11` and `scylla-2025.1` have no vector type, a lightweight
transaction fails on a ScyllaDB with tablets, and ScyllaDB refuses the type hint
`(text)NULL`. The values of `scylla://` that `dbrun` prints for ScyllaDB are
rewritten to `cassandra://`, because the driver has the one name.

A counter, a tuple and a user defined type are not in the round trip. A counter
column takes no `INSERT`, gocql `v2.1.2` cannot bind a tuple, and a user defined
type needs a type that `TestIntegrationTypes` makes first.
