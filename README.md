<div align="center">
  <a href="#about" title="About">About</a> |
  <a href="#installing" title="Installing">Installing</a> |
  <a href="#using" title="Using">Using</a> |
  <a href="#dsn" title="DSN">DSN</a> |
  <a href="#statements" title="Statements">Statements</a> |
  <a href="#scanning" title="Scanning">Scanning</a> |
  <a href="#related-projects" title="Related Projects">Related Projects</a> |
  <a href="#testing" title="Testing">Testing</a>
</div>

<br/>

[![Unit Tests][cassandra-ci-status]][cassandra-ci]
[![Go Reference][goref-cassandra-status]][goref-cassandra]

[cassandra-ci]: https://github.com/xo/cassandra/actions/workflows/test.yml "Test CI"
[cassandra-ci-status]: https://github.com/xo/cassandra/actions/workflows/test.yml/badge.svg "Test CI"
[goref-cassandra]: https://pkg.go.dev/github.com/xo/cassandra "Go Reference"
[goref-cassandra-status]: https://pkg.go.dev/badge/github.com/xo/cassandra.svg "Go Reference"

# About

`cassandra` is a Go [`database/sql`][database/sql] driver for
[Apache Cassandra][cassandra-db] and [ScyllaDB][scylladb]. It wraps
[the Apache Cassandra Go driver][gocql], which this file calls gocql, and it
registers the one name `cassandra`.

The driver is part of the [`xo`][xo] family of projects. [`usql`][usql] uses it
to connect to Cassandra and ScyllaDB, [`dburl`][dburl] writes its DSN, and
[`dbmeta`][dbmeta] reads database metadata through it. Its types, options and
errors follow the drivers of [`dbimp`][dbimp].

The driver was `github.com/xo/cql`. The name CQL is gone, because CQL is the
language and `cassandra` is the product. CQL is the query language of
Cassandra, and it looks like SQL.

# Installing

Install the driver in the usual way:

```sh
go get github.com/xo/cassandra@latest
```

# Using

Import the package for its side effect, and open a database with the name
`cassandra` and a [DSN](#dsn). A DSN is the text that names the server and the
settings for it. This program writes a row, reads it back, and reads a row that
holds a NULL:

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/cassandra"
)

func main() {
	ctx := context.Background()
	db, err := sql.Open("cassandra", "cassandra://cassandra:cassandra@127.0.0.1:9042/app?consistency=localQuorum")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// An argument that is a statement option changes how this one statement
	// runs. It is not sent to Cassandra as a value.
	id := uuid.NewV7()
	_, err = db.ExecContext(ctx,
		"INSERT INTO users (id, name, tags) VALUES (?, ?, ?)",
		id, "Ada", []string{"admin", "dev"},
		cassandra.WithConsistency(gocql.LocalQuorum), cassandra.WithTimeout(5*time.Second))
	if err != nil {
		log.Fatal(err)
	}

	// A column that can be NULL scans into a sql.Null[T]. A list scans into a
	// slice, and a uuid scans into the standard uuid.UUID.
	var (
		got  uuid.UUID
		name sql.Null[string]
		tags []string
	)
	err = db.QueryRowContext(ctx, "SELECT id, name, tags FROM users WHERE id = ?", id).Scan(&got, &name, &tags)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(got, name.V, tags)

	// Read many rows in pages of 500 rows.
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM users", cassandra.WithPageSize(500))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id   uuid.UUID
			name sql.Null[string]
		)
		if err := rows.Scan(&id, &name); err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name.Valid, name.V)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
```

All the connections of one `sql.DB` share one gocql session. The session keeps
its own pool of connections to each node. The pool settings of `sql.DB` limit
how many statements run at the same time, and the `numConns` key limits the
connections to each node.

To set a gocql option that the DSN cannot express, such as a host selection
policy or a logger, build a `gocql.ClusterConfig` and open it with a connector:

```go
cfg := gocql.NewCluster("10.0.0.1", "10.0.0.2")
cfg.Logger = logger
db := sql.OpenDB(cassandra.NewConnector(cfg))
```

# DSN

A DSN is a URL whose scheme is `cassandra`. The driver reads no other scheme and
no other form:

```text
cassandra://user:password@host1:9042/keyspace?consistency=localQuorum&host=host2:9042
```

The user information holds the credentials, the host part holds one host, and
the path names the keyspace. Each `host` key adds one more host, in order. A host
can be an IPv6 address, such as `cassandra://[::1]:9042/ks?host=[::2]:9042`.
[`dburl`][dburl] turns an alias, such as `scylla://` or `cql://`, into this URL,
so every alias works in `usql`.

The query takes these keys, and each has the default of `gocql.NewCluster`:

| Key | Value |
| --- | --- |
| `consistency` | `any`, `one`, `two`, `three`, `quorum`, `all`, `localQuorum`, `eachQuorum` or `localOne` |
| `keyspace` | the keyspace, when the path is empty |
| `timeout` | the timeout of a request, such as `10s` |
| `connectTimeout` | the timeout of a new connection, such as `10s` |
| `numConns` | the number of connections to each host |
| `ignorePeerAddr` | `true` or `false` |
| `disableInitialHostLookup` | `true` or `false` |
| `writeCoalesceWaitTime` | a duration, such as `200µs` |
| `username`, `password` | the credentials, when the user information holds none |
| `enableHostVerification` | `true` or `false`. Any TLS key turns TLS on. |
| `certPath`, `keyPath`, `caPath` | the paths of the TLS files |
| `host` | one more host. This is the one key that can repeat. |

An unknown key is an error that wraps `dbimp.ErrUnknownKey`. A scheme other than
`cassandra` wraps `dbimp.ErrScheme`. A key that appears twice, and a setting in
two places, such as a keyspace in the path and in a `keyspace` key, are errors
too. A DSN with no host connects to `127.0.0.1`. An error never holds the DSN,
because the DSN can hold a password.

# Statements

A statement takes its values at `?` markers. The driver passes every value that
gocql can marshal: slices, maps, `gocql.UUID`, `gocql.Duration`, `*big.Int`,
`net.IP`, a struct for a user defined type, and `gocql.UnsetValue`. It also takes
the types that a column returns: a `dbimp.Date`, a `dbimp.LocalTime`, a
`dbimp.Interval`, an `*apd.Decimal`, a `netip.Addr`, a `dbimp.Vector` and the
standard `uuid.UUID`. A collection of `uuid.UUID`, such as `[]uuid.UUID`, is not
supported. Use `[]gocql.UUID`.

A statement option changes how one statement runs. Pass it among the arguments,
or attach it to a context. An argument overrides the context, and the context
overrides the DSN:

```go
rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE org = ?", org, cassandra.WithPageSize(500))

ctx = cassandra.WithOptions(ctx, cassandra.WithConsistency(gocql.LocalOne))
```

| Option | Effect |
| --- | --- |
| `WithConsistency` | the consistency level |
| `WithSerialConsistency` | the serial consistency level of a lightweight transaction |
| `WithPageSize` | how many rows gocql reads in one page |
| `WithIdempotent` | whether gocql can run the statement more than once |
| `WithTimestamp` | the write time, in microseconds |
| `WithTimeout` | the time that the whole statement can take, over every page |
| `WithDatabase` | the keyspace of the statement. It needs protocol version 5. |
| `WithReadonly` | `WithReadonly(true)` fails, because Cassandra has no read-only statement |
| `WithParameter` | fails, because the protocol has no body of keys |

An option that Cassandra cannot honor fails the statement with an error that
wraps `dbimp.ErrNotSupported`, so a caller never believes that a limit holds when
it does not.

CQL has no transactions, so `BeginTx` returns an error that wraps
`dbimp.ErrNotSupported`. A batch is one CQL statement. Pass the whole
`BEGIN BATCH ... APPLY BATCH` text to `ExecContext`. To read the `[applied]`
column of a lightweight transaction, run it with `QueryContext`.

`ExecContext` returns `driver.ResultNoRows`, because Cassandra reports no row
count and no insert ID. A named argument from `sql.Named` is an error, because
gocql cannot bind a value by name. gocql `v2.1.2` cannot bind a tuple, so write
a tuple as a CQL literal.

# Scanning

Scan a column into any type that gocql can decode it into, such as `*[]string`,
`*map[string]int`, `*gocql.UUID` or `*time.Time`, or into any type that
`database/sql` can convert to. A `uuid` column also scans into `*uuid.UUID`,
`**uuid.UUID` and `sql.Null[uuid.UUID]`. A `*any` receives these Go types, which
are the kinds of [`dbimp`][dbimp]:

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
pointer to a pointer, or `*any`. Cassandra stores an empty collection as NULL, so
a NULL scanned into a slice or a map gives nil and no error.

# Related Projects

The driver works with these projects of the [`xo`][xo] family:

| Project | What it does |
| --- | --- |
| [`usql`][usql] | A command line client for many databases. It uses this driver for Cassandra and ScyllaDB. |
| [`dburl`][dburl] | Parses a database URL, and turns each alias, such as `scylla://`, into the URL that this driver reads. |
| [`dbmeta`][dbmeta] | Reads the metadata of many databases, and holds `dbrun`, which starts the test servers. |
| [`dbimp`][dbimp] | Other `database/sql` drivers, and the types, options and errors that this driver shares with them. |
| [`dbtpl`][dbtpl] | Generates code from a database schema, through `dbmeta`. |
| [`xo`][xo] | The home of the family of projects. |

# Testing

The unit tests need no server:

```sh
go test -race ./...
```

The integration tests run against the server that `CASSANDRA_DSN` names, and they
skip when it is empty. `CASSANDRA_ORDINARY_DSN` names an ordinary user, and
without it the test for that user skips. [CONTRIBUTING.md](CONTRIBUTING.md) shows
how to start a server with `dbrun`.

# Documents

| Document | Holds |
| --- | --- |
| [docs/CASSANDRA.md](docs/CASSANDRA.md) | what is known about Cassandra and ScyllaDB, and the type table |
| [docs/DESIGN.md](docs/DESIGN.md) | the design of the driver |
| [docs/PLAN.md](docs/PLAN.md) | the purpose, what exists, the testing plan and the open questions |
| [docs/decisions/](docs/decisions/README.md) | every decision, one file each |
| [docs/PROGRESS.md](docs/PROGRESS.md) | where the work stands |
| [docs/BACKLOG.md](docs/BACKLOG.md) | the planned work |
| [AGENTS.md](AGENTS.md) | the rules for a coding agent |
| [CONTRIBUTING.md](CONTRIBUTING.md) | the rules for a person |

# License

MIT. `cassandra` began as a fork of
[MichaelS11/go-cql-driver](https://github.com/MichaelS11/go-cql-driver), and
[LICENSE](LICENSE) keeps its notice.

[database/sql]: https://pkg.go.dev/database/sql
[cassandra-db]: https://cassandra.apache.org
[scylladb]: https://www.scylladb.com
[gocql]: https://github.com/apache/cassandra-gocql-driver
[xo]: https://github.com/xo
[usql]: https://github.com/xo/usql
[dburl]: https://github.com/xo/dburl
[dbmeta]: https://github.com/xo/dbmeta
[dbimp]: https://github.com/xo/dbimp
[dbtpl]: https://github.com/xo/dbtpl
