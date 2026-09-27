# cql backlog

This file records planned work for `cql`. Each item names the files,
decisions and issue numbers it touches, so that the work can start without
rediscovering the context.

This file is the source of truth for the roadmap. Issues are disabled on both
this repository and upstream, so nothing else holds one.

## How to refer to an item

Every item has an identifier of the form `W` followed by a number. Three rules
govern it:

1. Identifiers are append only. A new item takes the next unused number.
2. An identifier is never reused. When W1 is finished, no later item becomes
   W1.
3. An identifier is never renumbered. Removing W6 does not turn W7 into W6.

An item that is finished or abandoned keeps its heading and gains a status in
that heading. It does not disappear. A heading with no status is open. The
statuses in use are `Done`, `Dropped` and `Superseded by Wn`.

Decisions are a separate series, `D1`, `D2` and so on, in
[PLAN.md](PLAN.md). Do not confuse the two where both appear in one commit
message.

## Where the items came from

W1 to W17 bring the repository up to Go 1.27.1 and the `xo` layout, and fix
the defects found by reading the code on 2026-09-27. They are in the order
they must be done. W1 to W4 come first, because they set up the layout, the
CI and the tests that every later item uses.

W18 came from what the tests of W16 found in gocql.

The upstream sweep, done on 2026-09-27, found nothing new to import:

- `MichaelS11/go-cql-driver` is archived and has issues disabled. The API
  answers HTTP 410 for every issue number, so no upstream issue can be read.
  Commit `bd730e9` says "relates to #3", and that issue is gone with the rest.
- Upstream has three pull requests, #1, #2 and #4, and all three are merged
  and in `HEAD`: the `keyspace` option, the SSL options, and `go.mod`.
- Upstream has four forks. `xo/cql` and `xo/cqlsql` have no commits that
  upstream lacks. `bonedaddy/go-cql-driver` has none either.
  `choonkeat/go-cql-driver` is nine commits ahead, and all nine are the
  unsquashed history of pull request #2, which is already merged.

The defects come from the projects that use this driver:

- `xo/usql` #58, "Cassandra not working", closed. A statement came back as
  `(0 rows)` where `cqlsh` returned a row, and Ken diagnosed the driver as
  not reporting errors. That is W6.
- `xo/dbmeta` D62, in `dbmeta/docs/PLAN.md`. The driver returns an empty
  string for a CQL NULL, and a refused statement arrived as an empty result.
  Those are W7 and W6.
- `xo/usql` #154, "Expanded Test Suite", asked for Cassandra in CI. It lives
  on as W16 in `usql/docs/BACKLOG.md`, and W4 here is this repository's part
  of it.

The last section holds items that people raised and that have no priority yet.

## W1. Write CLAUDE.md, CONTRIBUTING.md and README.md. Done.

Done on 2026-09-27, with the rewrite. `docs_test.go` holds the rule for the
root documents.

Write the three root documents in the shape D3 sets, from
`dbmeta/CLAUDE.md`, `dbmeta/CONTRIBUTING.md` and `dbmeta/README.md`.

`CLAUDE.md` needs the "Which document to read" table, the hard rules, the Go
conventions from D7, the lint rule from D13, and the commands to run before a
commit:

```bash
gofmt -l . && go vet ./... && go build ./... && go test -race -count=2 ./...
golangci-lint run ./...
```

`README.md` loses the upstream GoDoc, Travis, Codecov and Go Report Card
badges, which all point at `MichaelS11/go-cql-driver`. Its note that tells callers to
read the error from `rows.Close` becomes wrong once W6 lands, so it goes at the same time.

Add `docs_test.go`, copied in spirit from `dbmeta/docs_test.go`. It fails on
Markdown at the root other than the three, and on a file in `docs/` that the
table in `CLAUDE.md` does not name.

`CONTRIBUTING.md` exists already, with the sections for writing text and for
agent skills (D15 and D16). Keep them. `CLAUDE.md` must tell an agent to load
the `simple-english` skill before it writes any text that a person reads, and
its table must name `CONTRIBUTING.md` for adding or updating a skill.

## W2. Move to github.com/xo/cql and Go 1.27.1. Done.

Done on 2026-09-27, as part of W16. `go.mod` names `github.com/xo/cql`, and
the Go 1.10 files are gone with the rest of the old code.

Do D1 and D2 in one commit.

- `go.mod`: `module github.com/xo/cql`, then `go mod tidy`. The `go 1.27.1`
  line is in already, because `skills_test.go` needs generics (D15).
- Remove the `// +build go1.10` lines from `cqlGo110.go`, `connector.go`,
  `cqlGo110_test.go` and `exampleSqlGo110_test.go`, and merge each file into
  the one it was split from.
- Replace `io/ioutil` with `io` in `cql.go`, `connector.go` and
  `cql_test.go`, and `interface{}` with `any` everywhere.

This stays on `gocql/gocql` so the diff stays mechanical. W5 moves the
dependency.

## W3. Replace Travis with GitHub Actions and add golangci-lint. Done.

Done on 2026-09-27. `.golangci.yml` and `.github/workflows/test.yml` came
with W16. The first run on GitHub, run 36287363683 for `v0.1.0`, passed the
unit, lint and releases jobs, and ran every integration test against
Cassandra 3.11 and 5.0. ScyllaDB joins the matrix when `dbmeta` pushes its
ScyllaDB releases and the pin in the workflow moves (D21).

The run warned that `actions/checkout@v4` and `actions/setup-go@v5` target
Node.js 20, which GitHub has deprecated. The workflow now uses `v7` of both,
which run on Node.js 24. The releases job now keys its cache on the
`go.sum` of `dbmeta`, because it checks out no other module.

`.travis.yml` is already staged for deletion in the working tree.

Add `.github/workflows/test.yml` in the shape of
`dburl/.github/workflows/test.yml`: one test job and one lint job, both on
`ubuntu-latest`, with `go-version-file: go.mod`, and `golangci-lint` pinned.

Add `.golangci.yml` with `version: "2"` and `default: all`, per D13. Start
from `dburl/.golangci.yml` and give every disabled linter its reason. Add
`coverage.out` to the root `.gitignore`, which is the only `.gitignore`.

The integration jobs get a server as D20 sets out: check out `xo/dbmeta`
at a pinned commit and start each release with `dbrun`. The release list
comes from `dbrun list --json` (D21). Set `DBMETA_RUNNER=docker`.

## W4. Split the tests into unit and integration tests. Done.

Done on 2026-09-27, as part of W16. The unit tests need no server and run
in parallel. `integration_test.go` reads `CQL_DSN`, skips when it is empty,
and creates one keyspace for each test, named for the time, with
`NetworkTopologyStrategy` in the data center of the node. `TestParseDSN`
parses both forms that `dbrun` prints (D20). The old flag tests are gone.

Do D11.

- Remove the `flag` parsing from `TestMain` in `cql_test.go`, along with the
  package level test variables it fills: the valid host, the invalid host,
  `EnableAuthentication`, `Username`, `Password` and the timeouts.
- Read `CQL_DSN`. A test that needs a server calls `t.Skip` when it is empty.
- `config_test.go` already needs no server. Make sure it runs, and that it
  runs in parallel.
- The invalid host tests dial `169.254.200.200` and wait for a timeout. Keep
  one of them and make it fast, or replace them with a closed local port.
- The destructive tests create and drop a keyspace named `cqltest` and a
  table named for the current time. Keep that, but name the keyspace per run
  so that two runs cannot collide.

Add a test that the DSN from `dbrun dsn` parses (D20). Measure the result
against Cassandra 3.11 and 5.0 with `dbrun` before you close this item.

## W5. Move to apache/cassandra-gocql-driver/v2. Done.

Done on 2026-09-27, as part of W16. The driver uses `v2.1.2`.

Do D6. Do W17 first, because `xo/cqlsql` has already done most of this work
and the work must not happen twice (D18).

Replace `github.com/gocql/gocql` with
`github.com/apache/cassandra-gocql-driver/v2` at `v2.1.2`. Compare every
`gocql` identifier in `config.go`, `globals.go` and `utility.go` with the
`v2` API. The four `ClusterConfig` fields that `config.go` sets are all still
there.

## W6. Return every error to the caller. Done.

Done on 2026-09-27, as part of W16. `TestErrorsReachTheCaller` and
`TestIntegrationErrors` cover a syntax error, a missing table, a missing
keyspace and a wrong password, against the fake and against real servers.

Do D9. Source: `xo/usql` #58, `xo/dbmeta` D62.

- `Ping` in `connection.go` returns `driver.ErrBadConn` for every failure and
  logs the cause. Return the cause. Wrap it in `ErrBadConn` only where a new
  connection can succeed, and never for an authentication or keyspace
  error.
- `queryContext` in `statement.go` returns rows and does not read the
  error from the iterator. Call `RowData` there, so that a refused statement fails from
  `QueryContext` itself.
- `Next` in `rows.go` returns `io.EOF` when `Scan` returns false. Return the
  iterator's `Close` error when there is one. Go 1.27 does pass a close
  error on through `rows.Err`, but only after the caller has read every row,
  and the result looks empty until then.
- `Open` and `OpenConnector` wrap with `%v` and name the function in the
  message. Use `%w` and a lower case gerund, per D7.

Start with integration tests for a statement with a syntax error, a table
that does not exist, a wrong password, and a keyspace that does not exist.
Confirm that each one fails before the fix. After it, each must return the
server's error from the first call that can.

## W7. Report a CQL NULL as nil. Done.

Done on 2026-09-27, as part of W16. `SELECT (text)NULL` scans into an
invalid `sql.Null[string]` on Cassandra 3.11 and 5.0. `dbmeta` can drop its
`pad` scanner once W14 lands.

Do D10. Source: `xo/dbmeta` D62 and `dbmeta/docs/NULLS.md`.

`Next` in `rows.go` scans into `iter.RowData().Values`, which are pointers to
values, and `interfaceToValue` in `utility.go` dereferences them. A NULL
arrives as the zero value. Scan into pointers to pointers instead, and pass
`nil` to `database/sql` when the inner pointer is nil.

Start with a test that selects `(text)NULL` and a NULL of every other CQL
type, scans each into `sql.Null[T]`, and expects `Valid` to be false. Then
tell `dbmeta` that it can remove the `pad` scanner in its Cassandra model.

## W8. Stop storing a context. Done.

Done on 2026-09-27, as part of W16.

Do the context part of D7.

`cqlConnStruct` stores `context` from `Connect`, and `Prepare` reuses it
later. That context belongs to the call that opened the connection, and it
can be cancelled long before the connection is used again. `Open` stores
`context.Background()`.

Remove the field. `database/sql` calls `PrepareContext`, `ExecContext` and
`QueryContext` whenever the driver has them, so the methods with no context
exist only to satisfy the interfaces.

## W9. Remove package level state. Done.

Done on 2026-09-27, as part of W16. The consistency table is unexported,
the driver logs nothing, and `FormatDSN` returns an error where the old code
panicked.

Do D8.

- `CqlDriver` in `globals.go` becomes an unexported value, and its `Logger`
  goes. So do the `Logger` fields on `CqlConnector` and `cqlConnStruct`, and
  the default logger `NewConnector` builds in `connector.go`.
- `DbConsistencyLevels` and `DbConsistency` become unexported.
- `ClusterConfigToConfigString` in `config.go` panics on a consistency it does
  not know. Return an error.
- Replace the `var` errors in `globals.go` with constants of `type Error
  string`, per D7.

## W10. Rename the exported types. Done.

Done on 2026-09-27, as part of W16, with the amendment in D17: `Connector`
keeps its configuration in an unexported field.

Do D12, using the table in `PLAN.md`. Name each receiver with one or two
letters. Do W9 and W10 together, because both touch every exported name.

## W11. Pass every gocql type through as a bind value. Done.

Done on 2026-09-27, as part of W16, with one exception that the driver
cannot correct: gocql `v2.1.2` cannot bind a tuple. See W18.

`CqlStmt.ColumnConverter` runs every argument through
`driver.DefaultParameterConverter`, which rejects any slice other than
`[]byte`, any map, and any array. `gocql.UUID` is a `[16]byte` with no `Value`
method, so a UUID cannot be bound either. The
`uint64` case in `statement.go` is a reflection workaround for one of them.

`driver.ColumnConverter` is deprecated. Implement `driver.NamedValueChecker`
on the connection and the statement, and let gocql's own marshalling decide
what it accepts. Test a `list`, a `set`, a `map`, a `uuid`, a `duration` and
a `varint` bound as arguments.

## W12. Report column types. Done.

Done on 2026-09-27, as part of W16.

`cqlRowsStruct` implements `Columns` and nothing else. Implement
`RowsColumnTypeDatabaseTypeName` and `RowsColumnTypeScanType` from
`iter.Columns()`, and `RowsColumnTypeNullable` answering unknown, because CQL
has no NOT NULL. `usql` and `dbmeta` both read column types through
`database/sql`.

## W13. Compare each DSN key with the driver. Done.

Done on 2026-09-27, as part of W16. All 14 keys set a field that
`ClusterConfig` in `v2.1.2` still has. `TestParseDSN`, `TestFormatDSNRoundTrip`
and `FuzzParseDSN` round trip them. The comment on `ParseDSN` lists every key,
and so does `README.md`.

`config.go` accepts 14 keys and rejects anything else. After W5, compare each one
with the `ClusterConfig` of `v2`. Write a table driven test that round trips
every key through `ConfigStringToClusterConfig` and
`ClusterConfigToConfigString`. Record in `README.md` which keys exist, since
`dburl` passes every unknown query parameter straight through (D5).

## W14. Point dburl, usql and dbmeta at github.com/xo/cql

Do this after the first tag, which is `v0.1.0` (D19).

- `dburl/scheme.go`: `GoPackage` and `DriverURL` for the `cql` scheme name
  `github.com/MichaelS11/go-cql-driver`.
- `usql/go.mod` pins `github.com/MichaelS11/go-cql-driver v0.1.1` and
  `github.com/gocql/gocql v1.7.0`.
- `usql/drivers/cassandra/cassandra.go` imports the old path, and sets
  `gocql.Logger` and `cql.CqlDriver.Logger`. Neither survives D6 and D8.
- `dbmeta/test` uses the same driver as `usql` (dbmeta D52), so it moves in
  the same release. Its `pad` workaround can go once W7 is in the tag.
  `dbmeta/test/parity_users_test.go` rewrites the credentials of a DSN in
  the D5 form, and `dbmeta/test/cassandra_test.go` names that form. Both
  must accept the URL form of D24.
- `dburl.GenCassandra` sends the URL itself as the DSN, with the scheme
  rewritten to `cql://` (D24). It stops writing the D5 form.

Each change is made in its own repository by that repository's own rules.

## W15. Share one gocql session per connector. Done.

Done on 2026-09-27, as part of W16.

D17 answers this: one session for each `Connector`. W16 builds it.

`Ping` in `connection.go` creates a `gocql.Session` for every
`database/sql` connection, lazily, on the first `Ping` or `PrepareContext`.
So `Connect` never fails, and the first failure surfaces later as
`ErrBadConn` (W6). Whatever the answer to the question, create the session in
`Connect` and return its error there.

## W16. Rewrite the driver to docs/DESIGN.md. Done.

Done on 2026-09-27. "What W16 measured" in `docs/DESIGN.md` records what
the tests found. The four open points of the design are settled there.

Do D17, in this repository (D18). It needs W2, W5 and W17 first.

The rewrite closes W6 to W12 and W15. Each of those items names a fault in
the current code, and each fault must have a test that fails before the
rewrite and passes after it. Build the files in the order of the Layout table
in `docs/DESIGN.md`, with the unit tests for each file in the same commit.

Settle the four points under "Open points for W16" at the end of
`docs/DESIGN.md`, each with a test. Record what each test found in
`docs/DESIGN.md`.

## W17. Bring over the work in xo/cqlsql. Done.

Done on 2026-09-27. Nothing was brought over. The working tree of
`xo/cqlsql` holds file renames, `io` in place of `io/ioutil`, and the import
path of `v2`. W16 rewrote every file and made the move to `v2` itself. The
work also kept every fault that W6 to W12 name, and it kept the exported
`Logger` that D8 removes. Ken can archive `xo/cqlsql`.

Do this before W5, because of D18. `xo/cqlsql` has uncommitted work in its
working tree, at `../cqlsql`: 26 files, 733 lines added and 1016 removed.
Read every change. Bring over each one that this repository lacks and that
the decisions here allow, as its own commit, and name `xo/cqlsql` in the
commit message. Record in this item what you took and what you left, with the
reason for each.

Do not change the `xo/cqlsql` working tree. When this item is done, tell Ken,
and he archives `xo/cqlsql`.

## W18. Report the tuple fault to apache/cassandra-gocql-driver

W16 found that gocql `v2.1.2` cannot bind a tuple. When a statement holds a
tuple marker, gocql counts each element of the tuple as a value of its own
(`actualColCount` in `frame.go`). Then it reads the type of each value from
the list of columns, which does not count the elements (`conn.go`, near the
`expected %d values send got %d` error). A statement fails, or it marshals a
value with the type of the wrong column.

W16 also found that `gocql.Unmarshal` into a `*[]any` panics when an element
of the tuple is NULL.

Filing an issue in another project is Ken's decision. Once he agrees, write
the report with a small program that fails, and link it here.

## W19. Remove gopkg.in/inf.v0 when gocql allows it

Do this when gocql changes how it handles a CQL `decimal`. Until then this
item waits (D26).

Today gocql `v2.1.2` decodes a `decimal` only into `*inf.Dec` and binds one
only from `inf.Dec`. When a release of gocql stops requiring `inf.Dec`, remove
every import of `gopkg.in/inf.v0` from this module: `types.go`, the tests, and
the `depguard` list in `.golangci.yml`. Then run `go mod tidy`, and remove
the entry from `go.mod` if gocql no longer requires the module. Read the
release notes of each gocql upgrade for this change.

## Raised, with no priority

- Decimals from another library. gocql binds a `decimal` column only from
  `inf.Dec` (D26). A decimal type whose `driver.Valuer` returns a string, as
  shopspring's does, therefore fails on a `decimal` column. The driver
  cannot see the column type when it checks an argument. A wrapper with
  `MarshalCQL` can, because gocql passes it the type of the column. That
  would let a string bind to a `decimal`. Measure the need before you build
  it.
- Named parameters. The driver refuses `sql.Named` with `ErrNamedArgs`
  (`docs/DESIGN.md`). CQL has `:name` bind markers, but gocql `v2.1.2` has no
  call that binds a value by name.
- Lightweight transactions. `docs/DESIGN.md` reads `[applied]` with
  `QueryRowContext`, and `ExecContext` discards it. `RowsAffected` can
  report it as 0 or 1 instead. Measure what `usql` shows before you choose.

