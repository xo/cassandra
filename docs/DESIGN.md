# Driver design

D30 to D33 amended this design on 2026-10-08: the DSN, the types, the options and
the errors follow `dbimp`. Where this document and those decisions differ, the
decisions win, and the sections below carry the change.

This document is the target design for the rewritten `cassandra` driver. Ken
accepted it on 2026-09-27, as D17 in [decisions/](decisions/README.md) records. D18 to D24
settle the questions that it left open. The work that builds it is W16 in
[BACKLOG.md](BACKLOG.md).

The driver is a `database/sql` driver for Apache Cassandra. It wraps
`github.com/apache/cassandra-gocql-driver/v2` (D6), which this document calls
gocql. It targets Go 1.27.1 (D2).

The design came from four sources, on 2026-09-27:

- The Go 1.27.1 source of `database/sql` and `database/sql/driver`.
- The gocql `v2.1.2` source.
- Answers from Gemini 3.1 Pro and DeepSeek V4. The last section says where
  they agreed, where they disagreed, and which claims were wrong.
- The `n1ql` driver, which is the same work for Couchbase. The two drivers
  share one file layout (n1ql D14).

Every claim about an API in this document was read in the source. A claim
that still needs a test says so.

## Goals

1. Implement every `database/sql/driver` interface that Cassandra can honor.
   Refuse the rest with a clear error.
2. Return every error that the server or gocql reports (D9).
3. Report a CQL NULL as NULL (D10).
4. Accept every Go value that gocql can bind, including collections, UUIDs
   and user defined types.
5. Hold no package level configuration (D8). Log nothing.
6. Run every unit test with no server. Gate the integration tests on
   `CASSANDRA_DSN` (D11).
7. Keep each file small, with one job, so that a developer or a coding agent
   can find a behavior by its file name.

## What database/sql gets

The table lists every interface in `database/sql/driver` for Go 1.27.1.

| Interface | Type | What the driver does |
| --- | --- | --- |
| `Driver` | `Driver` | `Open` parses the DSN and returns a connection. `database/sql` does not call it, because the driver implements `DriverContext`. |
| `DriverContext` | `Driver` | `OpenConnector` parses the DSN once and returns a `*Connector`. |
| `Connector` | `*Connector` | `Connect` returns a light `conn` over the shared session. |
| `io.Closer` | `*Connector` | `Close` closes the shared session. `sql.DB.Close` calls it. |
| `Conn` | `conn` | `Close` returns nil, because the session belongs to the connector. |
| `ConnPrepareContext` | `conn` | Returns a `stmt` that holds the statement text. |
| `ExecerContext` | `conn` | Runs a statement with no prepare step. |
| `QueryerContext` | `conn` | Runs a query with no prepare step. |
| `ConnBeginTx` | `conn` | Returns an error that wraps `dbimp.ErrNotSupported`. |
| `Pinger` | `conn` | Reads `release_version` from `system.local`. |
| `SessionResetter` | `conn` | Returns `driver.ErrBadConn` after the connector closes, and nil before. |
| `Validator` | `conn` | Returns false after the connector closes. |
| `NamedValueChecker` | `conn` | Decides which arguments go to gocql unchanged. See Arguments. |
| `Stmt` | `stmt` | `NumInput` returns -1. `Close` returns nil. |
| `StmtExecContext`, `StmtQueryContext` | `stmt` | Call the same code as the `conn` methods. |
| `Rows` | `rows` | `Columns`, `Close`, and `Next` for callers that use the driver directly. |
| `RowsColumnScanner` | `rows` | New in Go 1.27. Scans each column straight into the destination of the caller. See Rows. |
| `RowsColumnTypeDatabaseTypeName` | `rows` | The CQL type in upper case, such as `LIST<INT>`. |
| `RowsColumnTypeScanType` | `rows` | The Go type that `*any` receives for the column. |
| `RowsColumnTypeNullable` | `rows` | Reports unknown. See Column types. |
| `RowsColumnTypeLength` | not implemented | CQL types have no declared length. |
| `RowsColumnTypePrecisionScale` | not implemented | `decimal` and `varint` have no fixed precision. |
| `RowsNextResultSet` | not implemented | A CQL request holds one statement. |
| `Result` | `driver.ResultNoRows` | CQL reports no row count and no insert ID. |
| `Tx` | not implemented | CQL has no transactions. |
| `ColumnConverter` | not implemented | It is deprecated. `NamedValueChecker` replaces it. |
| `Execer`, `Queryer` | not implemented | They are deprecated. The context forms replace them. |

## Layout

The layout is the one that `n1ql` D14 proposes. A file that this driver does
not need is left out, and no file is added only to match.

| File | Holds |
| --- | --- |
| `driver.go` | `Driver`, `Open`, `OpenConnector`, and the `init` that calls `sql.Register("cql", ...)` (D4). |
| `connector.go` | `Connector`, `NewConnector`, the session life cycle, and `Close`. |
| `conn.go` | `conn` and every method that `database/sql` calls on it. |
| `stmt.go` | `stmt`. |
| `rows.go` | `rows`, `capture`, `NextRow`, `ScanColumn`, `Next`, and the column type methods. |
| `types.go` | The CQL type table: the canonical Go value, the scan type and the type name for each CQL type. |
| `options.go` | The query option types, such as `PageSize`, and `WithOptions`. |
| `dsn.go` | `ParseDSN` and `FormatDSN`. They replace `ConfigStringToClusterConfig` and `ClusterConfigToConfigString`. |
| `errors.go` | `type Error string`, the sentinel errors, and how a gocql error is wrapped. |
| `session.go` | The two unexported interfaces over gocql, and their adapters. See Testing. |

There is no `result.go`, because the driver returns `driver.ResultNoRows`.

## Connector and session

A gocql `Session` is a pool. It holds its own connections to every node and a
control connection. So the driver creates one session for each `Connector`,
and every `conn` shares it. The current driver creates one session for each
`database/sql` connection, and a pool of ten opens ten pools. D17 settles
this.

`clickhouse-go` and the `pgx` stdlib adapter take the same approach. The
native client owns the pool, and each `database/sql` connection is a handle.

```go
// Connector opens connections to one Cassandra cluster. All connections
// share one gocql session. Close the Connector to close the session.
type Connector struct {
	cfg gocql.ClusterConfig // a copy, so that the caller can change theirs

	mu      sync.Mutex
	sess    session // nil until the first Connect
	closed  bool
	newSess func(gocql.ClusterConfig) (session, error) // a fake in tests
}

// NewConnector returns a Connector for cfg. It does not connect.
func NewConnector(cfg *gocql.ClusterConfig) *Connector
```

`Connect` creates the session on the first call and returns the real error
from gocql if that fails. A failed attempt does not stay cached, so the next
`Connect` tries again. For that reason the code uses a mutex and not
`sync.Once`. After `Close`, `Connect` returns `ErrConnectorClosed`.

`sql.Open` and `OpenConnector` must not connect, by the convention of every
major driver. The first network traffic happens on the first `Connect`.

gocql `CreateSession` takes no context. The `connectTimeout` key in the DSN
limits how long it runs. `Connect` does not store its context (D7).

A caller that needs a gocql feature that the DSN cannot express builds a
`ClusterConfig`, sets it, and calls `sql.OpenDB(cassandra.NewConnector(cfg))`.
Examples are a host selection policy, a retry policy, a `Logger`, a
`QueryObserver` or a `ConnectObserver`. That is also the only way to log: the
driver itself logs nothing (D8).

`database/sql` pool settings such as `SetMaxOpenConns` now limit how many
statements run at once. They do not limit connections to Cassandra. gocql
`NumConns` limits those. The package documentation must say so.

## Conn

```go
type conn struct {
	c *Connector
}
```

A `conn` holds no state of its own. Cassandra has no state per connection
that `database/sql` can see. gocql refuses `USE` with `ErrUseStmt`, so a
statement cannot change the keyspace for later statements.

`ResetSession` and `IsValid` answer from the connector. When the connector is
closed, `ResetSession` returns `driver.ErrBadConn`, which is what the
interface asks for, and `IsValid` returns false. Otherwise they return nil
and true.

`Begin` and `BeginTx` return an error that wraps `dbimp.ErrNotSupported`.

## Statements

A `stmt` holds its `conn` and the statement text. gocql prepares statements
itself and caches them per session. gocql `v2` has no public call that
prepares a statement without running it. So `PrepareContext` sends nothing,
and a syntax error arrives from the first `ExecContext` or `QueryContext`.

`NumInput` returns -1. The driver cannot count `?` markers without parsing
CQL, because a `?` can appear inside a string literal.

The `conn` implements `ExecerContext` and `QueryerContext`, so `db.Exec` and
`db.Query` do not create a `stmt`. The `go-sql-driver/mysql` and `pgx`
drivers do the same.

## Arguments

`database/sql` asks `conn.CheckNamedValue` about each argument, in order.
When a driver implements `NamedValueChecker`, `database/sql` does not call
`driver.Valuer` by itself. It does that only when the checker returns
`driver.ErrSkip`. The rules, in order:

| The argument is | CheckNamedValue returns | Why |
| --- | --- | --- |
| named, from `sql.Named` | an error that wraps `dbimp.ErrNotSupported` | gocql `v2` has no named binding. |
| a query option, such as `WithPageSize` | nil | `ExecContext` and `QueryContext` take it out. See Query options. |
| nil | nil | gocql binds NULL. |
| a `gocql.Marshaler` | nil | gocql marshals it. This comes before `driver.Valuer`, so that a type with both uses the native form. |
| a `driver.Valuer`, such as `sql.Null[T]` | `driver.ErrSkip` | `database/sql` calls `Value` and converts the result. |
| a type of `dbimp`, an `*apd.Decimal` or a `netip.Addr` | nil, after it becomes the value that gocql takes | The canonical types of a column bind too (D32). |
| a channel, a function, a complex number or an unsafe pointer | an error that wraps `dbimp.ErrNotSupported` | gocql can never bind these. |
| anything else | nil | gocql marshals it when the statement runs. |

The last row accepts slices, maps, arrays, structs for user defined types,
`gocql.UUID`, `gocql.Duration`, `*big.Int`, `net.IP`,
`time.Time`, `time.Duration` and `gocql.UnsetValue`. A value that does not fit
the column fails when the statement runs. The error from gocql names the
column type and the Go type, and the driver returns it wrapped. The driver
cannot know the column types before the statement runs, because
`PrepareContext` sends nothing.

The current driver runs every argument through
`driver.DefaultParameterConverter`, which refuses all of these.

## Query options

A query option changes how gocql runs one statement. The options are in
`options.go`, and `Option` is `dbimp.Option[options]` (D33 and dbimp D109):

| Function | Effect |
| --- | --- |
| `WithConsistency` | `Query.Consistency` |
| `WithSerialConsistency` | `Query.SerialConsistency` |
| `WithPageSize` | `Query.PageSize` |
| `WithIdempotent` | `Query.Idempotent` |
| `WithTimestamp` | `Query.WithTimestamp`, in microseconds |
| `WithTimeout` | a deadline on the context of the statement, over every page of its rows |
| `WithDatabase` | `Query.SetKeyspace`, which needs protocol version 5 |
| `WithReadonly` | `WithReadonly(true)` fails with `dbimp.ErrNotSupported` |
| `WithParameter` | fails with `dbimp.ErrNotSupported`, because the protocol has no body of keys |

A caller can pass an option in two ways (D23). The first is an argument:

```go
rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE org = ?",
	org, cassandra.WithConsistency(gocql.LocalQuorum), cassandra.WithPageSize(500))
```

The second is a context, which carries the option to every call that uses it:

```go
ctx = cassandra.WithOptions(ctx, cassandra.WithConsistency(gocql.LocalQuorum))
```

The DSN sets the defaults for the whole session, such as `consistency`, and
gocql applies them to every query it creates. For each statement, the driver
calls `dbimp.Resolve`, which applies the options from the context and then the
options from the arguments. So the order is the DSN, then the context, then the
argument, and each one overrides the one before it. An option can be anywhere in
the list of arguments, and `Resolve` numbers the values again. The `conn` stores
nothing between calls.

## Rows

### Scanning

Go 1.27 added `driver.RowsColumnScanner` (golang/go#67546). When `rows`
implements it, `database/sql` does not call `Next`. It calls `NextRow` once per
row and then `ScanColumn` once per column, with the destination that the
caller passed to `Scan`. The driver can then let gocql decode straight into a
`*[]string`, a `*map[string]int` or a `*gocql.UUID`. The current driver cannot
do this.

gocql has no public call that returns the bytes of one column. But gocql
calls `UnmarshalCQL(info, data)` on any destination that implements
`gocql.Unmarshaler`, and it passes nil data for NULL. So `NextRow` scans the
row into one `capture` per column:

```go
// capture holds one column of the current row, as gocql read it.
type capture struct {
	info gocql.TypeInfo
	null bool   // true when the column is NULL
	data []byte // the column in the CQL wire format, reused from row to row
}

// UnmarshalCQL satisfies gocql.Unmarshaler.
func (c *capture) UnmarshalCQL(info gocql.TypeInfo, data []byte) error {
	c.info, c.null = info, data == nil
	c.data = append(c.data[:0], data...) // gocql reuses its frame buffer
	return nil
}
```

The copy is necessary. gocql `readBytes` returns a slice of the frame buffer.
`null` is a separate field, because `append` to an empty slice returns nil,
and an empty value is not NULL.

`ScanColumn` then picks one of two paths by the type of the destination:

| The destination is | Path |
| --- | --- |
| a `sql.Scanner`, such as `sql.Null[T]` or `sql.NullString` | database/sql: decode to the canonical value, then call `sql.ConvertAssign`. |
| `*any` | database/sql |
| `*sql.RawBytes` | database/sql. The bytes are the text form of the canonical value, never the wire format. |
| anything else | gocql: call `gocql.Unmarshal(info, data, dest)`. |

If gocql refuses a destination that is a pointer to a basic kind, such as
`*float64` for an `int` column, `ScanColumn` tries the database/sql path. A
basic kind is a bool, an integer, a float or a string. The second try is
safe only if gocql writes nothing to a scalar that it refuses. A test must
prove that for each basic kind. This keeps
`var f float64; rows.Scan(&f)` working, as it does with `lib/pq` and `mysql`.
The driver does not retry for a slice, a map or a struct, because gocql can
write part of one before it fails. DeepSeek raised that risk.

`sql.ConvertAssign` gets the `ScanContext` from `ScanColumn`, as its
documentation asks.

`Next` stays for code that uses the driver without `database/sql`. It fills
each slot with the canonical value.

### NULL

| The destination is | A NULL gives |
| --- | --- |
| a `sql.Scanner` | `Scan(nil)`, so `sql.Null[T]` has `Valid` false |
| `*any` or `*sql.RawBytes` | nil |
| a pointer to a pointer, such as `**string` | the inner pointer set to nil |
| `*[]T` or `*map[K]V` | a nil slice or map, and no error |
| a pointer to a basic kind, such as `*string` | the error `converting NULL to string is unsupported` from `sql.ConvertAssign` |
| a pointer to a struct, such as `*time.Time` or `*gocql.UUID` | an error from `sql.ConvertAssign` |

A collection gets no error, because Cassandra stores an empty collection as
NULL. gocql sets nil for the pointer, slice and map rows, as its `Unmarshal`
source shows.

The last two rows are a breaking change. Today the driver writes the zero
value. The new behavior is what `lib/pq` and `go-sql-driver/mysql` do, and it
is what D10 asks for. W14 must find every caller in `usql` and `dbmeta` that
depends on the old behavior.

### Canonical values

The canonical value is what `*any` receives, what a `sql.Scanner` receives,
what `Next` returns, and what `ColumnTypeScanType` reports. One CQL type has
one canonical Go type in all four places, by the kinds of `dbimp` (D32).

| CQL type | Canonical Go type |
| --- | --- |
| `ascii`, `text`, `varchar` | `string` |
| `blob` | `[]byte` |
| `boolean` | `bool` |
| `tinyint`, `smallint`, `int`, `bigint`, `counter` | `int64` |
| `float`, `double` | `float64` |
| `varint` | `*big.Int` |
| `decimal` | `*apd.Decimal` |
| `timestamp` | `time.Time`, in UTC |
| `date` | `dbimp.Date` |
| `time` | `dbimp.LocalTime` |
| `duration` | `dbimp.Interval` |
| `uuid`, `timeuuid` | `uuid.UUID`, from the standard library (D25) |
| `inet` | `netip.Addr` |
| `list<T>`, `set<T>`, `tuple<...>` | `[]any`, of the canonical values of the elements |
| `map<text, V>`, a user defined type | `map[string]any` |
| `map<K, V>` with another key | the map that gocql chooses |
| `vector<T, N>` of numbers | `dbimp.Vector[T]` |

`canonical` asks gocql to decode the value into the Go type that gocql chooses,
and `normalize` turns it into the canonical value, element by element for a
collection. A `gocql.Unmarshaler` that is not a `uuid.UUID` still decodes into
the type that gocql chooses.

`CheckNamedValue` converts each canonical type that gocql does not take as an
argument: a `uuid.UUID` to a `gocql.UUID`, a `dbimp.Date`, an `*apd.Decimal`
and the other types of `dbimp` to a value that writes its own wire form. gocql
writes a `time.Time` that is the zero value as an empty value, so a date is
never a `time.Time` on the way in.

A caller that wants `gocql.UUID` or a type of gocql scans into that type, and
gocql decodes it.

Unit tests pin every row of this table.

## Column types

`ColumnTypeDatabaseTypeName` builds the name from `TypeInfo.Type()` and, for
a collection, from its `Key` and `Elem`. It cannot use `String()`, because
gocql writes a collection as `map(text, int)`, which is not CQL. The name is
in upper case, as the `database/sql/driver` documentation asks: `TEXT`,
`BIGINT`, `LIST<INT>`, `MAP<TEXT, INT>`. The name of a user defined type
keeps its case, because a quoted CQL name is case sensitive.

`ColumnTypeScanType` returns the canonical Go type from the table above.

`ColumnTypeNullable` returns `ok` false. A regular column can be NULL and a
primary key column cannot. gocql `ColumnInfo` does not say which a column is.
Reading the schema for every result costs a round trip, and the answer can be
stale. Gemini proposed `(true, true)`, which is wrong for a key column.

## Results

`ExecContext` returns `driver.ResultNoRows`. Its `RowsAffected` and
`LastInsertId` return an error. Cassandra reports neither number. Both models
proposed this.

A lightweight transaction is a statement with `IF`, such as
`INSERT ... IF NOT EXISTS`. Its result is a row with an `[applied]` column.
To read it, call `QueryRowContext` and scan the first column into a `bool`.
`ExecContext` discards that row. The package documentation must show an
example.

## Transactions and batches

`BeginTx` returns an error that wraps `dbimp.ErrNotSupported`. The driver does not emulate a
transaction with a `BATCH`. A logged batch is atomic, but it is not isolated,
it cannot roll back, and a query inside it cannot read its own writes. Both
models agreed.

A batch is a CQL statement. To run one, pass its text to `ExecContext`:

```go
_, err := db.ExecContext(ctx, `BEGIN BATCH
	INSERT INTO users (id, name) VALUES (?, ?);
	INSERT INTO users_by_name (name, id) VALUES (?, ?);
APPLY BATCH`, id, name, name, id)
```

The package documentation must show this example.

## Errors

`errors.go` holds the error type and the one sentinel error of the driver
(D7). The other errors are those of `dbimp` (D33):

```go
// Error is an error that the driver reports.
type Error string

// Error values.
const (
	// ErrConnectorClosed is returned by Connect after Close.
	ErrConnectorClosed Error = "connector is closed"
)
```

A DSN wraps `dbimp.ErrScheme`, `dbimp.ErrUnknownKey`, `dbimp.ErrRepeatedKey` or
`dbimp.ErrInvalidValue`. `BeginTx`, a named argument, an argument that gocql
cannot bind, and a call that takes no context wrap `dbimp.ErrNotSupported`. A
result that fails after a row reached the caller wraps `dbimp.ErrIncomplete`.

The driver returns every gocql error wrapped with `%w`. So
`errors.As` finds a `gocql.RequestError` and its concrete types, such as
`gocql.RequestErrSyntax` and `gocql.RequestErrReadTimeout`, and `errors.Is`
finds `gocql.ErrSessionClosed`. A test must confirm which of these gocql
returns as a pointer and which as a value.

The driver never returns `driver.ErrBadConn` from a statement.
`database/sql` answers `ErrBadConn` by trying the statement again on another
`conn`. Every `conn` shares one session, so the second try meets the same
fault. A retry can also run a write twice. gocql already retries inside its
own pool, under its retry policy. The one use of `ErrBadConn` is the one that
`SessionResetter` defines, after the connector closes.

`QueryContext` returns the error from the server itself (D9). gocql sends
the request and reads the first page when the iterator is created, so the
`query` adapter can report a refused statement before it returns. See The
seam.

`NextRow` returns the error from `Iter.Close` when the rows end with a fault,
and `io.EOF` when they end normally.

## Context

Every gocql call gets the context of the call that caused it, through
`Query.WithContext`. A cancelled context stops the request that runs, and it
stops each later page fetch. `rows` holds the context of its query, because
gocql fetches the next page during `NextRow`. No `conn` or `Connector` field
holds a context. The driver never calls `context.Background` or
`context.TODO`.

## DSN

A DSN is a URL whose scheme is `cassandra`, parsed with `net/url`. The driver
reads no other scheme and no older form (D31, and dbimp D27 and dbimp D35).
`dburl` writes the URL, and rewrites its alias schemes, such as `scylla://`, to
`cassandra://` first, so this driver never repeats the alias list.

```text
cassandra://user:password@h1:9042/keyspace?consistency=localQuorum&host=h2
cassandra://[::1]:9042/keyspace?host=[::2]:9042&host=h3
```

| Part of the URL | Configuration |
| --- | --- |
| user information | `username` and `password` |
| host | one host, with an optional port |
| path | the keyspace, as one segment |
| query | the keys of the README, and a `host` key that can repeat |

The host part holds one host, and each `host` key in the query adds one more, in
order (D34). A comma in the host part is an error. Any host can be an IPv6 address
in brackets. A setting that appears in two places is an error, such as a user name
in the user information and in a `username` key.

`dsn.go` reads the query with `dbimp.NewQuery`, which refuses a key that it does
not know and a key that repeats, and with the helpers of `dbimp.Query`, each with
the default of `gocql.NewCluster`. It holds two functions:

```go
// ParseDSN parses a DSN, which is a URL that starts with cassandra://.
func ParseDSN(dsn string) (*gocql.ClusterConfig, error)

// FormatDSN writes cfg as a cassandra:// URL. It returns an error for a value
// that has no DSN form, such as an unknown consistency.
func FormatDSN(cfg *gocql.ClusterConfig) (string, error)
```

`FormatDSN` returns an error where `ClusterConfigToConfigString` panics
today. A key is added only through W13.

## Testing

### The seam

The driver reaches gocql through two unexported interfaces in `session.go`.
Each has three methods, and each is defined where the driver consumes it:

```go
// session is the part of a gocql session that the driver uses.
type session interface {
	exec(ctx context.Context, stmt string, values []any, o options) error
	query(ctx context.Context, stmt string, values []any, o options) (iterator, error)
	close()
}

// iterator is the part of a gocql iterator that the driver uses.
type iterator interface {
	columns() []gocql.ColumnInfo
	scan(dest ...any) bool
	close() error
}
```

`session.go` also holds the two adapters over `*gocql.Session` and
`*gocql.Iter`. The `query` adapter calls `Iter.RowData` once before it
returns. `RowData` returns the error of the iterator and reads no row, so a
refused statement fails from `query`. They are the only code that calls gocql
to run a statement.

`fake_test.go` holds a fake `session`. It records each statement, its values
and its options, and it returns rows that a test sets up. The fake builds each
column with `gocql.Marshal` and passes the bytes to `UnmarshalCQL`, so the
decode path in a unit test is the real gocql code. `gocql.NewNativeType`
builds the type of any column. For a native type it takes the type code. For
any other type it takes the name, such as
`gocql.NewNativeType(4, gocql.TypeCustom, "map<text, int>")`, and parses it.
`TupleTypeInfo` and `UDTTypeInfo` have exported fields, so a test also builds
them directly.

### The tests

| File | What it tests | Server |
| --- | --- | --- |
| `dsn_test.go` | Both forms, every key, the `host` key with IPv6, every conflict, every error, and a round trip through `ParseDSN` and `FormatDSN`. `FuzzParseDSN` makes sure that no input panics and that a DSN that parses also round trips. | no |
| `conn_test.go` | Each rule in the Arguments table, the split of query options, options from the context, an argument that overrides the context, `BeginTx`, `ResetSession`, `IsValid`, and the wrapping of errors. | no |
| `rows_test.go` | Each row of the Scanning, NULL and Canonical values tables, and each column type method. | no |
| `connector_test.go` | One session for many `Connect` calls, a new try after a failure, and `Connect` after `Close`. | no |
| `skills_test.go` | D15. | no |
| `example_test.go` | `Example` functions for `sql.Open`, `NewConnector`, query options, a lightweight transaction and a batch. | no output checked |
| `integration_test.go` | Every CQL type in and out, every NULL rule, collections, paging, cancellation, and the errors in W6. | `CASSANDRA_DSN` |

The unit tests use table driven subtests, `t.Parallel`, and `t.Context()` in
place of `context.Background`. `BenchmarkScan` uses `b.Loop` and measures the
allocations of a scan per row. The race detector runs in CI with
`go test -race -count=2 ./...`, as in `dbmeta`.

## What came from which driver

| Choice | Taken from |
| --- | --- |
| One native pool per connector, and a light `conn` | `clickhouse-go`, `pgx` stdlib |
| `ExecerContext` and `QueryerContext` on the connection | `go-sql-driver/mysql`, `pgx` stdlib |
| `ParseDSN` and `FormatDSN` | `go-sql-driver/mysql` |
| Query options as typed arguments | `pgx` stdlib (`QueryExecMode`) |
| An error for NULL into a basic kind | `lib/pq`, `go-sql-driver/mysql`, through `sql.ConvertAssign` |
| Strings for UUID and numeric values in `*any` | `pgx` stdlib |
| `driver.ResultNoRows` when the server reports no count | both models, and the `driver` package itself |
| `RowsColumnScanner` | none yet. It is new in Go 1.27. |

## What the models said

Gemini 3.1 Pro answered all ten questions. DeepSeek V4 timed out three times
on the full list and then answered it in three parts.

They agreed on these points, and the design follows them:

- One session per connector, a `conn` with no state, and `Conn.Close` that
  returns nil.
- No `ErrBadConn` for query faults. DeepSeek added that a retry can run a
  write twice.
- `BeginTx` returns an error. No emulation over `BATCH`.
- `driver.ResultNoRows`. Read `[applied]` with a query.
- Copy the column bytes in `UnmarshalCQL`.
- Transparent paging, with a page size option.
- A small unexported seam over gocql, with a fake for unit tests.

They disagreed on these points:

- Query options. Gemini proposed typed arguments. DeepSeek proposed context
  values. Ken chose both, and an argument overrides the context (D23).
- Scan dispatch. Gemini proposed "try gocql, and fall back on any error".
  DeepSeek proposed a dispatch on the destination type, because gocql can
  write part of a slice or a struct before it fails. The design takes the
  dispatch and allows the fallback only for scalars.
- Nullable. Gemini proposed `(true, true)`. DeepSeek proposed `(false, true)`
  for key columns from the schema. The design reports unknown. See Column
  types.
- Statements. DeepSeek proposed that `stmt` hold a gocql prepared handle.
  gocql `v2` exposes none.

These claims from the models were wrong, and the design does not use them:

- `sql.ErrNotSupported` does not exist.
- `database/sql` does not wrap a `ScanType` to handle NULL.
- `sql.Null[T]` arrived in Go 1.22, not Go 1.26. `testing/synctest` arrived
  in Go 1.24 as an experiment and in Go 1.25 as a stable package, not in Go
  1.27.
- `ResetSession` returns an `error`, not a `bool`.
- `clickhouse-go` passes its settings through the context, not through
  `ErrRemoveArgument`.

## What W16 measured

W16 settled the four open points of this design with tests, on 2026-09-27.
It ran the unit tests, a fuzz test of the DSN, and the integration tests
against Cassandra 3.11 and 5.0 and ScyllaDB 2025.1 and 2026.3, each started
with `dbrun`.

1. `TypeInfo.Zero` reports `net.IP` for `inet`, `*big.Int` for `varint`,
   `*inf.Dec` for `decimal`, `int` for `int`, `float32` for `float`,
   `gocql.UUID` for `uuid`, `[]int` for `list<int>`, `map[string]int` for
   `map<text, int>`, `[]any` for a tuple and `map[string]any` for a user
   defined type. The table under Canonical values holds, and
   `TestScanCanonical` pins every row.
2. gocql returns each `RequestErr` type as a pointer. `errors.As` with a
   `*gocql.RequestErrSyntax` finds a syntax error from a real server.
3. `gocql.NewNativeType` builds a collection type from its name. See The
   seam.
4. gocql writes nothing to a scalar that it refuses, and the fallback in
   Scanning is safe. gocql does write part of a slice before it fails:
   `list<text>` into `*[]int` leaves `[]int{0, 0}`. So the driver does not
   fall back for a slice, a map or a struct.

The tests also found these facts. The design above already follows each one.

- gocql `v2.1.2` cannot bind a tuple. It counts each element of a tuple
  marker as a value of its own, and then it reads the column types without
  that count. A statement with a tuple marker fails with `expected 25 values
  send got 24`, or reads the wrong column type. The driver cannot correct
  this. W18 reports it.
- gocql decodes a tuple into `*[]any` with a panic when an element is NULL.
  So the driver builds the canonical value of a tuple itself, from the
  element captures.
- gocql reports a NULL tuple as a tuple whose elements are all NULL.
- gocql drops `frozen<...>` when it reads a type, and `String` on a
  `TypeInfo` returns no CQL name. `typeName` builds the name.
- `net/url` reads the text after the last colon of the host part as the
  port. So `cql://h1:9042,h2` fails with `invalid port ":9042,h2"`, and a
  list parses only when its last host has a port or when no host has one.
  `FormatDSN` writes a list only when `net/url` reads it back, and uses
  `host` keys otherwise. Open question 8 in `PLAN.md` asks whether the
  driver splits the host part itself.
- ScyllaDB refuses the type hint `(text)NULL`. ScyllaDB 2026.3 refuses
  `SimpleStrategy`, because it uses tablets. ScyllaDB 2025.1 refuses a
  lightweight transaction on a table with tablets. These are product
  differences, and the integration tests skip each one with the reason.
- `stmt.Exec` and `stmt.Query` take no context, and the driver never calls
  `context.Background`. So they return a new error, `ErrNoContext`.
  database/sql never calls them, because the driver has the forms that take
  a context.
- `Driver.Open` has no context either. It creates a `Connector` of its own,
  and the `Close` of that one connection closes its session.
