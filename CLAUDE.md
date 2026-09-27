# cql

`cql` is a `database/sql` driver for Apache Cassandra and ScyllaDB. It wraps
`github.com/apache/cassandra-gocql-driver/v2`, which this file calls gocql,
and it registers itself as `cql`. `usql` and `dbmeta` use it, and `dburl`
writes its DSN.

## Which document to read

| If you are | Read |
| --- | --- |
| changing how the driver behaves | [docs/DESIGN.md](docs/DESIGN.md), which is the design that the code follows |
| asking why something is the way it is | the table at the top of [docs/PLAN.md](docs/PLAN.md) |
| looking for the next piece of work | [docs/BACKLOG.md](docs/BACKLOG.md) |
| changing how a column is decoded | "Rows" in [docs/DESIGN.md](docs/DESIGN.md), then `types.go` and `rows.go` |
| changing how an argument is bound | "Arguments" in [docs/DESIGN.md](docs/DESIGN.md), then `conn.go` |
| adding a DSN key | the comment on `ParseDSN` in `dsn.go`, D5 and D24 |
| adding a query option | `options.go` and D23 |
| testing against a server | "Before you commit" below, D11 and D20 |
| answering a lint finding | "Linting" below and D13 |
| adding or updating an agent skill | [CONTRIBUTING.md](CONTRIBUTING.md), under Agent skills, and D15 |
| writing a document, a code comment, an error message or a commit message | the `simple-english` skill. Load it first. See D16 |

`CONTRIBUTING.md` holds the same material for a person, and it is shorter.
[README.md](README.md) is for a person who uses the driver.

A document that is not in this table does not exist. If you cannot find where
something is written down, it is not written down. Do not decide an open
question yourself. The open questions are at the end of `docs/PLAN.md`. Ask
Ken.

## Hard rules

1. The driver registers as `cql`, and the DSN has two forms: a `cql://` or
   `cassandra://` URL, and the older list of hosts with a query. `dburl`,
   `usql` and `dbmeta` depend on both. Never rename the driver, never drop a
   form, and never change what a DSN key means. See D4, D5 and D24.
2. The package holds no configuration and logs nothing. A caller who needs a
   gocql setting builds a `gocql.ClusterConfig` and passes it to
   `NewConnector`. See D8.
3. Every error reaches the caller, wrapped with `%w`. Never return
   `driver.ErrBadConn` from a statement: every connection shares one session,
   so a retry meets the same fault, and a retry can run a write twice. See D9.
4. A CQL NULL reaches `database/sql` as nil. Never hand it a zero value in
   place of NULL. See D10.
5. `context.Context` comes first and is never stored. The library never calls
   `context.Background` or `context.TODO`. See D7.
6. The module depends on the standard library, gocql and `gopkg.in/inf.v0`,
   and on nothing else. `depguard` holds the list. gocql brings in `inf.v0`,
   and it stays until gocql allows its removal, which is W19 (D26). Ask Ken
   before you add a dependency.
7. Only `session.go` calls gocql to run a statement. Every other file reaches
   gocql through the `session` and `iterator` interfaces, so that the unit
   tests can replace it with the fake in `fake_test.go`.
8. A unit test needs no server. A test that needs one reads `CQL_DSN` and
   skips when it is empty. See D11.
9. Never start a container by hand. `dbrun` in `dbmeta` starts every server,
   as D20 says. See "Before you commit".
10. Ken reviews the staged changes. Stage your work and stop. Commit or push
    only when Ken says so.

## Layout

| File | Holds |
| --- | --- |
| `driver.go` | The package documentation, `Driver`, and the `init` that registers `cql`. |
| `connector.go` | `Connector`, which owns the one gocql session. |
| `conn.go` | `conn`, and how an argument is checked. |
| `stmt.go` | `stmt`. |
| `rows.go` | `rows`, `capture`, and how a column is scanned. |
| `types.go` | The canonical Go value, the scan type and the name of each CQL type. |
| `options.go` | The query options and `WithOptions`. |
| `dsn.go` | `ParseDSN` and `FormatDSN`. |
| `errors.go` | `Error` and the sentinel errors. |
| `session.go` | The two interfaces over gocql, and their adapters. |
| `fake_test.go` | The fake session and iterator that the unit tests use. |
| `integration_test.go` | The tests that need a server. |
| `docs_test.go` | The rules for the documents, from D3. |
| `skills_test.go` | The rule for the agent skills, from D15. |

## Go conventions

Match the surrounding code. D7 holds the conventions, and these are the
ones that are easy to miss:

- A sentinel error is a constant of `type Error string` in `errors.go`.
  Compare errors with `errors.Is` and `errors.As`.
- An error message is lower case, starts with a gerund, and names what
  failed, such as `creating session: %w`. It does not say "failed to".
- A receiver is named with one or two lower case letters.
- Use `any`, `new(expr)`, `t.Context()` in tests, and `b.Loop()` in
  benchmarks.

## Linting

`.golangci.yml` holds the rules. The posture is `default: all` with a disable
list, and the reason for each entry is written beside it. The version is
pinned in `.github/workflows/test.yml`. See D13.

A linter that makes idiomatic Go worse is disabled, with the reason. Only a
real defect gets a code change. A change made only to quiet a linter is
itself a defect.

## Before you commit

Run these in the repository root. `gofmt -l .` must print nothing.

```bash
gofmt -l . && go vet ./... && go test -race -count=2 ./...
golangci-lint run ./...
```

To run the integration tests, start a server with `dbrun` from `dbmeta`, and
set `CQL_DSN` to the DSN that it prints:

```bash
(cd ../dbmeta/test && go run ./cmd/dbrun start cassandra-5.0)
export CQL_DSN=$(cd ../dbmeta/test && go run ./cmd/dbrun dsn --json cassandra-5.0 | jq -r '.[0].dsn')
go test -race -count=1 -run Integration ./...
```

CI runs the integration tests against every Cassandra and ScyllaDB release
that `dbmeta` tests (D21).

## Writing documentation

Load the `simple-english` skill before you write any text that a person
reads: a document, a code comment, an error message or a commit message.
Follow it for that text (D16). In short: short sentences, the active voice,
`can`, `will` and `must` in place of `should` and `may`, no contractions, no
semicolons, no em dashes, the condition before the command, and one word for
one meaning.
