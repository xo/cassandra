# cassandra

`cassandra` is a `database/sql` driver for Apache Cassandra and ScyllaDB. It wraps
`github.com/apache/cassandra-gocql-driver/v2`, which this file calls gocql,
and it registers itself as `cassandra`. `usql` and `dbmeta` use it, and `dburl`
writes its DSN.

## Standing rules

These hold in every `xo` repository, for every coding agent (D110 of `dbmeta`,
D28).

1. Stage changes for review. Commit and push only when Ken says so.
2. Load the `simple-english` skill before you write any text that a person
   reads: a document, a code comment, an error message or a commit message.
   Follow it for that text.
3. Load the `go-pedantry` skill before you write or review Go code. Follow it
   where it does not conflict with a rule in this file. A rule here wins.

`CLAUDE.md` holds one line that imports this file, so that Claude Code and
every other agent read the same rules. Edit this file, not that one.

## Hard rules

1. The driver registers one name with `database/sql`, `cassandra`, and no alias
   (dbimp D28 and dbimp D30). Its DSN is a standard URL whose scheme is that
   name, parsed with `net/url`, with no form kept from an earlier driver (D31).
   Never write a list of schemes or of aliases here. That taxonomy belongs to
   `dburl`, which writes the URL. Never rename the driver without Ken. See D27.
2. The package holds no configuration and logs nothing. A caller who needs a
   gocql setting builds a `gocql.ClusterConfig` and passes it to
   `NewConnector`. See D8.
3. A CQL NULL reaches `database/sql` as nil. Never hand it a zero value. A row
   keeps the column order of the statement. A value has the Go type that fits
   the CQL type of its column, by the kinds of `dbimp`, and a string column gives
   a string. See D10 and D32.
4. `context.Context` comes first, is named `ctx`, and is never stored in a
   struct. The library never calls `context.Background` or `context.TODO`. The
   driver implements the context forms of the `database/sql/driver` interfaces
   and stops work when the context ends. See D7.
5. Every error reaches the caller, wrapped with `%w`. Return
   `driver.ErrBadConn` only when the connection is broken and the statement did
   not reach the server. A statement never returns it, because every connection
   shares one session, so a retry meets the same fault, and a retry can run a
   write twice. See D9 and D33.
6. The module depends on the standard library, gocql, `github.com/xo/dbimp` and
   `github.com/cockroachdb/apd/v3` for decimals, and on nothing else.
   `depguard` holds the list. Ask Ken before you add a dependency. See D30.
7. Never import `dburl`, `dbmeta`, `usql` or `dbtpl`. Read their source when you
   need a fact, and run `dbrun` from a checkout of `dbmeta` as a tool. Only a
   test imports `dbimp/dbimptest`.
8. Never write a `//go:build` constraint on an operating system or an
   architecture, and never branch on either.
9. A unit test needs no server. A test that needs one reads `CASSANDRA_DSN` and
   skips when it is empty. See D11.
10. Never start a container by hand. `dbrun` in `dbmeta` starts every server,
    as D20 says. See "Running the checks".
11. Register the driver from `init` in `driver.go`, never from a test body.
    `go test -count=2` fails on a second registration, and that is why CI runs
    it.
12. Ken reviews the staged changes. Stage your work and stop. Commit, push, tag
    or publish a release only when Ken says so, and ask again for each one.
13. Only `session.go` calls gocql to run a statement. Every other file reaches
    gocql through the `session` and `iterator` interfaces, so that the unit
    tests can replace it with the fake in `fake_test.go`.

## Layout

| File | Holds |
| --- | --- |
| `driver.go` | The package documentation, `Driver`, and the `init` that registers `cassandra`. |
| `connector.go` | `Connector`, which owns the one gocql session. |
| `conn.go` | `conn`, and how an argument is checked. |
| `stmt.go` | `stmt`. |
| `rows.go` | `rows`, `capture`, and how a column is scanned. |
| `types.go` | The canonical Go value, the scan type and the name of each CQL type, and how an argument of a canonical type is bound. |
| `options.go` | The query options and `WithOptions`. |
| `dsn.go` | `ParseDSN` and `FormatDSN`. |
| `errors.go` | `Error` and `ErrConnectorClosed`. The other errors are those of `dbimp`. |
| `session.go` | The two interfaces over gocql, and their adapters. |
| `fake_test.go` | The fake session and iterator that the unit tests use. |
| `integration_test.go` | The tests that need a server. |
| `tables_test.go` | The type table and the interface table of `docs/CASSANDRA.md`, written from the code (D30). |
| `contract_test.go` | The contract that every `xo` driver keeps, run against the fake session (D30). |
| `docs_test.go` | The rules for the documents, from D3, D28 and D29. |
| `prose_test.go` | The rules of `simple-english` that a machine can check, from D16. |
| `skills_test.go` | The rule for the agent skills, from D15. |

## Which document to read

| If you are | Read |
| --- | --- |
| changing how the driver behaves | [docs/DESIGN.md](docs/DESIGN.md), which is the design that the code follows |
| asking why something is the way it is | the index in [docs/decisions/](docs/decisions/README.md) |
| asking what the project is for, or what is open | [docs/PLAN.md](docs/PLAN.md) |
| asking what is known about Cassandra and ScyllaDB, and the type table | [docs/CASSANDRA.md](docs/CASSANDRA.md) |
| resuming work after a session ended | [docs/PROGRESS.md](docs/PROGRESS.md) |
| looking for the next piece of work | [docs/BACKLOG.md](docs/BACKLOG.md) |
| changing how a column is decoded | "Rows" in [docs/DESIGN.md](docs/DESIGN.md), then `types.go` and `rows.go` |
| changing how an argument is bound | "Arguments" in [docs/DESIGN.md](docs/DESIGN.md), then `conn.go` |
| adding a DSN key | the comment on `ParseDSN` in `dsn.go`, D5 and D24 |
| adding a query option | `options.go` and D23 |
| testing against a server | "Running the checks" below, D11 and D20 |
| answering a lint finding | "Running the checks" below and D13 |
| adding or updating an agent skill | [CONTRIBUTING.md](CONTRIBUTING.md), under Agent skills, and D15 |
| writing a document, a code comment, an error message or a commit message | the `simple-english` skill. Load it first. See D16 |

`CONTRIBUTING.md` holds the same material for a person, and it is shorter.
[README.md](README.md) is for a person who uses the driver.

A document that is not in this table does not exist. If you cannot find where
something is written down, it is not written down. Do not decide an open
question yourself. The open questions are at the end of `docs/PLAN.md`. Ask
Ken.

## Decisions

`docs/decisions/` holds every decision, one file each, and
`docs/decisions/README.md` is the index of all 29. Read the status first,
because a decision can amend or replace an earlier one. A decision is never
edited to change its conclusion. A new decision gets the next number and a file
of its own, and its row in the index. See D29.

## Go conventions

Match the surrounding code. D7 holds the conventions, and these are the
ones that are easy to miss:

- A sentinel error is a constant of a defined string type, never a variable
  made with `errors.New`. Use the errors of `dbimp` where one fits. Compare
  errors with `errors.Is` and `errors.As`.
- An error message is lower case, starts with a gerund, and names what
  failed, such as `creating session: %w`. It does not say "failed to".
- A receiver is named with one or two lower case letters, and every method of a
  type uses the same one.
- Accept interfaces and return concrete types. Keep an interface to three
  methods or fewer, and define it where it is consumed.
- A nullable value is `sql.Null[T]`, a UUID is the standard `uuid.UUID` (D25),
  and a decimal is an `*apd.Decimal`. Read JSON, if the driver ever reads any,
  with `encoding/json/v2`, never with `encoding/json`.
- Use `any`, `new(expr)`, `t.Context()` in tests, and `b.Loop()` in
  benchmarks.

## Running the checks

Run these in the repository root before you stage the work. `gofmt -l .` must print nothing.

```bash
gofmt -l . && go vet ./... && go build ./... && go test -race -count=2 ./...
golangci-lint run ./...
```

To run the integration tests, start a server with `dbrun` from `dbmeta`, and
set `CASSANDRA_DSN` to the URL that it prints. Set `DBMETA_OWNER_NAME` to the name
of your session on every `dbrun` command:

```bash
(cd ../dbmeta/test && go run ./cmd/dbrun start cassandra-5.0)
export CASSANDRA_DSN=$(cd ../dbmeta/test && go run ./cmd/dbrun dsn --json cassandra-5.0 | jq -r '.[0].url')
go test -race -count=1 -run Integration ./...
```

CI runs the integration tests against every Cassandra and ScyllaDB release
that `dbmeta` tests (D21).

`.golangci.yml` holds the rules for `golangci-lint`. The posture is
`default: all` with a disable list, and the reason for each entry is written
beside it. The version is pinned in `.github/workflows/test.yml`. See D13.

A linter that makes idiomatic Go worse is disabled, with the reason. Only a
real defect gets a code change. A change made only to quiet a linter is
itself a defect.

## Skills

Two skills are committed as copies in `.agents/skills/` and `.claude/skills/`
(D15). Load `simple-english` before you write any text that a person reads
(D16). In short: short sentences, the active voice, `can`, `will` and `must`
in place of `should` and `may`, no contractions, no semicolons, no em dashes,
the condition before the command, and one word for one meaning.
`prose_test.go` checks the rules that a machine can check. Load `go-pedantry`
before you write or review Go code.
