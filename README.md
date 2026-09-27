# cql

`cql` is a Go `database/sql` driver for Apache Cassandra and ScyllaDB. It
wraps [the Apache Cassandra Go driver][gocql] (gocql) and registers itself as
`cql`.

[gocql]: https://github.com/apache/cassandra-gocql-driver

```bash
go get github.com/xo/cql
```

## Use

```go
import (
	"database/sql"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	_ "github.com/xo/cql"
)

db, err := sql.Open("cql", "cql://cassandra:cassandra@127.0.0.1:9042/app")
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
`sql.OpenDB(cql.NewConnector(cfg))`.

## DSN

A DSN has two forms. A DSN that starts with `cql://` or `cassandra://` is a
URL:

```text
cql://user:password@host1:9042,host2:9042/keyspace?consistency=localQuorum
```

The user information sets the credentials, and the path names the keyspace.
`net/url` refuses some lists of hosts: a list that holds an IPv6 address, and
a list where an earlier host has a port and the last one has none. For those,
add each host with a `host` key: `cql://[::1]:9042/ks?host=[::2]:9042`.

Any other DSN is a list of hosts, separated by commas, and a query. This is
the form that the driver has always read:

```text
host1:9042,host2?keyspace=keyspace&username=user&password=password
```

Both forms take these keys:

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
| `host` | one more host. The key can repeat. |

An unknown key is an error. So is a key that appears twice, and a setting in
two places, such as a keyspace in the path and in a `keyspace` key. A DSN
with no host connects to `127.0.0.1`.

## Arguments

A statement takes its values at `?` markers. The driver passes every value
that gocql can marshal: slices, maps, `gocql.UUID`, `gocql.Duration`,
`*big.Int`, `*inf.Dec`, `net.IP`, a struct for a user defined type, and
`gocql.UnsetValue`. The driver also takes the standard `uuid.UUID`, and
`sql.Null[uuid.UUID]`, and converts each to a `gocql.UUID`. A collection of
`uuid.UUID`, such as `[]uuid.UUID`, is not supported: use `[]gocql.UUID`. A `driver.Valuer`, such as `sql.Null[T]`, works as it
does with any driver.

A query option changes how one statement runs. Pass it among the arguments,
or attach it to a context. An argument overrides the context, and the
context overrides the DSN:

```go
rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE org = ?", org, cql.PageSize(500))

ctx = cql.WithOptions(ctx, cql.Consistency(gocql.LocalOne))
```

The options are `Consistency`, `SerialConsistency`, `PageSize`, `Idempotent`
and `Timestamp`.

## Scanning

Scan a column into any type that gocql can decode it into, such as
`*[]string`, `*map[string]int`, `*gocql.UUID` or `*time.Time`, or into any
type that `database/sql` can convert to. A `uuid` column also scans into
`*uuid.UUID`, `**uuid.UUID` and `sql.Null[uuid.UUID]`. `*any` receives these Go types:

| CQL type | Go type |
| --- | --- |
| `ascii`, `text`, `varchar` | `string` |
| `blob` | `[]byte` |
| `boolean` | `bool` |
| `tinyint`, `smallint`, `int`, `bigint`, `counter` | `int64` |
| `float`, `double` | `float64` |
| `varint`, `decimal`, `inet` | `string` |
| `uuid`, `timeuuid` | `uuid.UUID`, from the standard library |
| `timestamp`, `date` | `time.Time`, in UTC |
| `time` | `time.Duration` |
| `duration` | `gocql.Duration` |
| `list`, `set`, `map`, `vector` | the slice or map that gocql chooses, such as `[]int` |
| `tuple` | `[]any` |
| a user defined type | `map[string]any` |

A NULL scanned into a plain `*string` or `*int64` is an error, as it is with
other drivers. To read a column that can be NULL, scan into `sql.Null[T]`, a
pointer to a pointer, or `*any`. Cassandra stores an empty collection as
NULL, so a NULL scanned into a slice or a map gives nil and no error.

## Limits

- CQL has no transactions. `BeginTx` returns `cql.ErrNoTransactions`. A batch
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

The integration tests run against the server that `CQL_DSN` names, and skip
when it is empty. [CONTRIBUTING.md](CONTRIBUTING.md) shows how to start one.

## Design

| Document | Holds |
| --- | --- |
| [docs/DESIGN.md](docs/DESIGN.md) | the design of the driver |
| [docs/PLAN.md](docs/PLAN.md) | every decision, and the open questions |
| [docs/BACKLOG.md](docs/BACKLOG.md) | the planned work |
| [CLAUDE.md](CLAUDE.md) | the rules for a coding agent |
| [CONTRIBUTING.md](CONTRIBUTING.md) | the rules for a person |

## License

MIT. `cql` began as a fork of
[MichaelS11/go-cql-driver](https://github.com/MichaelS11/go-cql-driver), and
[LICENSE](LICENSE) keeps its notice.
