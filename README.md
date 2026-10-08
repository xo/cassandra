# cassandra

`cassandra` is a Go `database/sql` driver for Apache Cassandra and ScyllaDB. It
wraps [the Apache Cassandra Go driver][gocql] (gocql) and registers itself as
`cassandra`.

[gocql]: https://github.com/apache/cassandra-gocql-driver

```bash
go get github.com/xo/cassandra
```

## Use

```go
import (
	"database/sql"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	_ "github.com/xo/cassandra"
)

db, err := sql.Open("cassandra", "cassandra://cassandra:cassandra@127.0.0.1:9042/app")
if err != nil {
	return err
}
defer db.Close()

var (
	id   gocql.UUID
	name sql.Null[string]
	tags []string
)
err = db.QueryRowContext(ctx, "SELECT id, name, tags FROM users WHERE id = ?", uid).Scan(&id, &name, &tags)
```

All the connections of one `sql.DB` share one gocql session, which holds its
own pool of connections to each node. The pool settings of `sql.DB` limit how
many statements run at the same time. The `numConns` key limits the
connections to each node.

To set a gocql option that the DSN cannot express, such as a host selection
policy or a logger, build a `gocql.ClusterConfig` and open it with
`sql.OpenDB(cassandra.NewConnector(cfg))`.

## DSN

A DSN is a URL whose scheme is `cassandra`. The driver reads no other scheme
and no other form (D31):

```text
cassandra://user:password@host1:9042/keyspace?consistency=localQuorum&host=host2:9042
```

The user information sets the credentials, the host part holds one host, and the
path names the keyspace. Each `host` key adds one more host, in order, and any
host can be an IPv6 address: `cassandra://[::1]:9042/ks?host=[::2]:9042` (D34).

The query takes these keys, and each has the default of `gocql.NewCluster`:

| Key | Value |
| --- | --- |
| `consistency` | `any`, `one`, `two`, `three`, `quorum`, `all`, `localQuorum`, `eachQuorum` or `localOne` |
| `keyspace` | the keyspace |
| `timeout` | the timeout of a request, such as `10s` |
| `connectTimeout` | the timeout of a new connection, such as `10s` |
| `numConns` | the number of connections to each host |
| `ignorePeerAddr` | `true` or `false` |
| `disableInitialHostLookup` | `true` or `false` |
| `writeCoalesceWaitTime` | a duration, such as `200µs` |
| `username`, `password` | the credentials |
| `enableHostVerification` | `true` or `false`. Any TLS key turns TLS on. |
| `certPath`, `keyPath`, `caPath` | the paths of the TLS files |
| `host` | one more host. This is the one key that can repeat. |

An unknown key is an error that wraps `dbimp.ErrUnknownKey`, and a scheme other
than `cassandra` wraps `dbimp.ErrScheme`. A key that appears twice, and a
setting in two places, such as a keyspace in the path and in a `keyspace` key,
are errors too. A DSN with no host connects to `127.0.0.1`.

## Arguments

A statement takes its values at `?` markers. The driver passes every value
that gocql can marshal: slices, maps, `gocql.UUID`, `gocql.Duration`,
`*big.Int`, `net.IP`, a struct for a user defined type, and `gocql.UnsetValue`.
It also takes the types that a column returns: a `dbimp.Date`, a
`dbimp.LocalTime`, a `dbimp.Interval`, an `*apd.Decimal`, a `netip.Addr`, a
`dbimp.Vector` and the standard `uuid.UUID`. A collection of `uuid.UUID`, such
as `[]uuid.UUID`, is not supported: use `[]gocql.UUID`. A `driver.Valuer`, such
as `sql.Null[T]`, works as it does with any driver.

A query option changes how one statement runs. Pass it among the arguments,
or attach it to a context. An argument overrides the context, and the
context overrides the DSN:

```go
rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE org = ?", org, cassandra.WithPageSize(500))

ctx = cassandra.WithOptions(ctx, cassandra.WithConsistency(gocql.LocalOne))
```

The options are `WithConsistency`, `WithSerialConsistency`, `WithPageSize`,
`WithIdempotent` and `WithTimestamp`, and the four that every
`xo` driver takes (dbimp D109): `WithTimeout`, `WithReadonly`, `WithParameter`
and `WithDatabase`. Cassandra has no read-only statement and no body of keys,
so `WithReadonly(true)` and `WithParameter` fail the statement with an error
that wraps `dbimp.ErrNotSupported`.

## Scanning

Scan a column into any type that gocql can decode it into, such as
`*[]string`, `*map[string]int`, `*gocql.UUID` or `*time.Time`, or into any
type that `database/sql` can convert to. A `uuid` column also scans into
`*uuid.UUID`, `**uuid.UUID` and `sql.Null[uuid.UUID]`. `*any` receives these Go
types, which are the kinds of `dbimp` (dbimp D135 and D32 here):

| CQL type | Go type |
| --- | --- |
| `ascii`, `text`, `varchar` | `string` |
| `blob` | `[]byte` |
| `boolean` | `bool` |
| `tinyint`, `smallint`, `int`, `bigint`, `counter` | `int64` |
| `float`, `double` | `float64` |
| `decimal` | `*apd.Decimal` |
| `varint` | `*big.Int` |
| `inet` | `netip.Addr` |
| `uuid`, `timeuuid` | `uuid.UUID`, from the standard library |
| `timestamp` | `time.Time`, in UTC |
| `date` | `dbimp.Date` |
| `time` | `dbimp.LocalTime` |
| `duration` | `dbimp.Interval` |
| `list`, `set`, `tuple` | `[]any`, of the Go types of its elements |
| `map` with a text key, a user defined type | `map[string]any` |
| `vector` of numbers | `dbimp.Vector[T]` |

[docs/CASSANDRA.md](docs/CASSANDRA.md) holds the whole table, with the scan type
and the database type of each.

A NULL scanned into a plain `*string` or `*int64` is an error, as it is with
other drivers. To read a column that can be NULL, scan into `sql.Null[T]`, a
pointer to a pointer, or `*any`. Cassandra stores an empty collection as
NULL, so a NULL scanned into a slice or a map gives nil and no error.

## Limits

- CQL has no transactions. `BeginTx` returns an error that wraps `dbimp.ErrNotSupported`. A batch
  is one CQL statement: pass the whole `BEGIN BATCH ... APPLY BATCH` text to
  `ExecContext`.
- `ExecContext` returns `driver.ResultNoRows`, because Cassandra reports no
  row count and no insert ID. To read `[applied]` from a lightweight
  transaction, run it with `QueryContext`.
- Named arguments from `sql.Named` are refused, because gocql cannot bind a
  value by name.
- gocql `v2.1.2` cannot bind a tuple value. Write a tuple as a CQL literal.

## Testing

The unit tests need no server:

```bash
go test -race ./...
```

The integration tests run against the server that `CASSANDRA_DSN` names, and skip
when it is empty. [CONTRIBUTING.md](CONTRIBUTING.md) shows how to start one.

## Design

| Document | Holds |
| --- | --- |
| [docs/DESIGN.md](docs/DESIGN.md) | the design of the driver |
| [docs/CASSANDRA.md](docs/CASSANDRA.md) | what is known about Cassandra and ScyllaDB, and the type table |
| [docs/PLAN.md](docs/PLAN.md) | the purpose, what exists, the testing plan and the open questions |
| [docs/decisions/](docs/decisions/README.md) | every decision, one file each |
| [docs/PROGRESS.md](docs/PROGRESS.md) | where the work stands |
| [docs/BACKLOG.md](docs/BACKLOG.md) | the planned work |
| [AGENTS.md](AGENTS.md) | the rules for a coding agent |
| [CONTRIBUTING.md](CONTRIBUTING.md) | the rules for a person |

## License

MIT. `cassandra` began as a fork of
[MichaelS11/go-cql-driver](https://github.com/MichaelS11/go-cql-driver), and
[LICENSE](LICENSE) keeps its notice.
