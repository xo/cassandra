# Decisions

Every decision that shapes this repository, in the order it was made. Entries
are append only. A decision is never edited to change its conclusion. When a
later decision replaces one, the older entry keeps its text and gains a line
at the top saying what replaced it.

Each heading carries a status. `Decided` means Ken has settled it or it
follows from a rule every `xo` repository keeps. `Proposed` means it is a
recommendation that waits for Ken. An amendment must be visible from both
sides: if D11 amends D4, then D4 says so too.

Work items live in [BACKLOG.md](BACKLOG.md) and are numbered `W1`, `W2` and
so on. Decisions are numbered `D1`, `D2` and so on. The two series never mix.

| Decision | Title | Status |
| --- | --- | --- |
| [D1](#d1-the-module-path-is-githubcomxocql-decided) | The module path is github.com/xo/cql | Decided |
| [D2](#d2-the-go-directive-is-1271-decided) | The go directive is 1.27.1 | Decided |
| [D3](#d3-the-repository-uses-the-xo-layout-decided) | The repository uses the xo layout | Decided |
| [D4](#d4-the-driver-registers-as-cql-decided) | The driver registers as cql | Decided |
| [D5](#d5-the-dsn-is-the-one-dburl-writes-decided-amended-by-d24) | The DSN is the one dburl writes | Decided. Amended by D24 |
| [D6](#d6-the-driver-moves-to-apachecassandra-gocql-driverv2-decided) | The driver moves to apache/cassandra-gocql-driver/v2 | Decided |
| [D7](#d7-go-conventions-are-the-ones-dburl-and-dbmeta-keep-decided) | Go conventions are the ones dburl and dbmeta keep | Decided |
| [D8](#d8-the-package-holds-no-configuration-decided) | The package holds no configuration | Decided |
| [D9](#d9-an-error-always-reaches-the-caller-decided) | An error always reaches the caller | Decided |
| [D10](#d10-a-cql-null-reaches-databasesql-as-nil-decided) | A CQL NULL reaches database/sql as nil | Decided |
| [D11](#d11-a-test-that-needs-a-server-reads-cql_dsn-decided) | A test that needs a server reads CQL_DSN | Decided |
| [D12](#d12-exported-names-do-not-stutter-decided-amended-by-d17) | Exported names do not stutter | Decided. Amended by D17 |
| [D13](#d13-golangci-lint-runs-in-ci-at-a-pinned-version-decided) | golangci-lint runs in CI at a pinned version | Decided |
| [D14](#d14-the-licence-stays-as-upstream-wrote-it-decided) | The licence stays as upstream wrote it | Decided |
| [D15](#d15-agent-skills-are-committed-as-copies-decided) | Agent skills are committed as copies | Decided |
| [D16](#d16-every-text-a-person-reads-follows-simple-english-decided) | Every text a person reads follows simple-english | Decided |
| [D17](#d17-the-driver-is-rewritten-to-docsdesignmd-decided-amends-d12) | The driver is rewritten to docs/DESIGN.md | Decided. Amends D12 |
| [D18](#d18-this-repository-carries-the-modern-driver-decided) | This repository carries the modern driver | Decided |
| [D19](#d19-the-first-tag-is-v010-decided) | The first tag is v0.1.0 | Decided |
| [D20](#d20-ci-starts-cassandra-with-dbrun-from-a-pinned-dbmeta-commit-decided) | CI starts Cassandra with dbrun from a pinned dbmeta commit | Decided |
| [D21](#d21-the-tested-releases-are-the-ones-dbmeta-tests-decided) | The tested releases are the ones dbmeta tests | Decided |
| [D22](#d22-the-duration-helpers-are-removed-decided) | The duration helpers are removed | Decided |
| [D23](#d23-a-query-option-can-come-from-an-argument-or-from-the-context-decided) | A query option can come from an argument or from the context | Decided |
| [D24](#d24-the-dsn-can-be-a-url-decided-amends-d5) | The DSN can be a URL | Decided. Amends D5 |

### D1. The module path is github.com/xo/cql. Decided.

`go.mod` still says `github.com/MichaelS11/go-cql-driver`. Upstream is
archived on GitHub and last took a change on 2020-09-20. Every `xo` module is
named `github.com/xo/<repository>`.

The new path is a breaking change for every importer, so it is the one chance
to make the other breaking changes in this file. `usql`, `dbmeta` and `dburl`
all name the old path, and W14 moves them.

### D2. The go directive is 1.27.1. Decided.

`go.mod` says `go 1.27.1` and has no `toolchain` line. That is Ken's target
and what the other `xo` repositories use. The line changed first, together
with `go mod tidy`, because `skills_test.go` needs generics (D15). The `// +build go1.10` constraints
and the files split out for them go, because no supported Go needs them.

### D3. The repository uses the xo layout. Decided.

This matches `dbmeta` and `usql`:

- The root holds `README.md`, `CLAUDE.md` and `CONTRIBUTING.md`, and no other
  Markdown. Every other document is in `docs/`.
- `CLAUDE.md` follows the order `dbmeta/CLAUDE.md` uses: purpose, a "Which
  document to read" table, numbered hard rules with their reasons, layout, Go
  conventions, linting, the commands to run before committing, and how to
  write documentation.
- `CONTRIBUTING.md` holds the same material for a person, and it is shorter.
- `docs/PLAN.md` is this file. It holds decisions only.
- `docs/BACKLOG.md` holds work items, in the format `usql/docs/BACKLOG.md`
  uses.
- The package stays flat, with one root package named `cql`. A sub-package is
  added only where there is a real boundary.
- Examples are `Example` tests. Golden files go in `testdata/`.
- CI is one GitHub Actions workflow, `.github/workflows/test.yml`, on
  `ubuntu-latest`, with `go-version-file: go.mod`. There is no Makefile.

`dbmeta` has a test, `docs_test.go`, that fails on a stray Markdown file at the
root or a document missing from the table in `CLAUDE.md`. This repository gets
the same test, in W1.

### D4. The driver registers as cql. Decided.

`sql.Register("cql", ...)` stays. `dburl` maps its `cql` scheme and the aliases
`ca`, `cassandra`, `datastax`, `scy` and `scylla` to the driver name `cql`
(`dburl/scheme.go`), and
`usql` and `dbmeta` open it by that name. A new name breaks all
three and gains nothing.

### D5. The DSN is the one dburl writes. Decided. Amended by D24.

Amended by D24: a DSN can also be a URL, and after the rewrite `dburl` sends
the URL. The form below stays accepted. Ken accepted this entry on
2026-09-27.

The DSN is `host[:port][,host[:port]...]?key=value&...`, and the keys are the
camel case names `config.go` accepts today. `dburl.GenCassandra` writes
exactly this shape. It sets `username`, `password` and `keyspace`, and passes
every other query parameter through.

`dburl` owns the connection string taxonomy. Changing the format here means
changing `GenCassandra` in the same release, and there is no reason to. A new
option is added as a new key, and an existing key never changes meaning.

### D6. The driver moves to apache/cassandra-gocql-driver/v2. Decided.

Ken accepted this on 2026-09-27.

`github.com/gocql/gocql` is now `github.com/apache/cassandra-gocql-driver/v2`.
This repository pins a `gocql/gocql` pseudo version from 2020-08-15.
`usql` pins `gocql/gocql v1.7.0`. The Apache module is at `v2.1.2`.

The move is a major version. Every `ClusterConfig` field that `config.go`
reads is still in `v2.1.2`: `IgnorePeerAddr`, `DisableInitialHostLookup`,
`WriteCoalesceWaitTime` and `SslOpts`. Nobody has compared the rest of the API
yet. `xo/cqlsql` already made this move in its working tree, so it is the
first place to look. See the first open question.

`v2` has no package level `gocql.Logger`. A logger is set on the
`ClusterConfig` instead, as `Logger StructuredLogger`. `usql` sets
`gocql.Logger` today (`usql/drivers/cassandra/cassandra.go`), so it stops
compiling when it moves to `v2`. W14 covers that.

### D7. Go conventions are the ones dburl and dbmeta keep. Decided.

- Sentinel errors are constants of `type Error string`, never variables built
  with `errors.New` or `fmt.Errorf`. An importer can reassign a variable but
  not a constant.
- Every error is wrapped with `%w`. A message is lower case, starts with a
  gerund, and names the object that failed. It does not say "failed to" or
  "error".
- `context.Context` comes first, is named `ctx`, and is never stored in a
  struct.
- `any` instead of `interface{}`, `io` instead of `io/ioutil`.
- A receiver is named with one or two lower case letters, the same letter on
  every method of a type.

`config.go`, `connection.go` and `globals.go` break each of these today.

### D8. The package holds no configuration. Decided.

Ken accepted this on 2026-09-27.

`CqlDriver` is an exported package variable with a mutable `Logger`, and it
logs to standard error by default. `DbConsistencyLevels` and `DbConsistency`
are exported maps that any importer can change. `dbmeta` hard rule 6 says why
this matters: one driver in `usql` borrowed another's configuration and
shipped a fault.

So the registered driver value is unexported. The consistency tables are
unexported. The driver logs nothing, and every fact it used to log goes into
the error it returns instead, which is D9. If a caller needs gocql's own log,
it sets `Logger` on the `ClusterConfig` it passes to the `Connector`. That
field exists in `v2` (D6).

`usql` writes `cql.CqlDriver.Logger` today. W14 removes that line.

### D9. An error always reaches the caller. Decided.

Two places lose an error today.

`Ping` replaces every failure with `driver.ErrBadConn`, and logs the cause to
standard error. A wrong password therefore arrives as "driver: bad
connection". `database/sql` retries on `ErrBadConn`, so a failure that cannot
succeed on retry is tried again. From here on, `ErrBadConn` is returned only
when a retry on a new connection can help, and it wraps the cause.
`database/sql` in Go 1.27 matches it with `errors.Is`, so a wrapped
`ErrBadConn` still triggers the retry.

`QueryContext` never returns the error from the statement. It builds the
iterator, reads its columns, and returns rows. The error arrives only once a
caller iterates. `usql` #58 and the Cassandra work in `dbmeta` (D62, "cost an
afternoon") both saw an empty result where the server had refused the
statement. From here on, `QueryContext` returns the server's error itself.

### D10. A CQL NULL reaches database/sql as nil. Decided.

gocql decodes a NULL of any type as the zero value of that type, and
`interfaceToValue` passes the zero value on. So `SELECT (text)NULL` scanned
into `sql.Null[string]` comes back `Valid` and `""`. `dbmeta` measured this,
and it works around it with a scanner that discards a padded column
(`dbmeta/docs/PLAN.md` D62, and the Cassandra section of
`dbmeta/docs/NULLS.md`). A real catalog column that is NULL cannot be rescued
there at all.

The driver fixes this by scanning each column into a pointer to a pointer,
so that gocql can leave the inner pointer nil for a NULL. W7 does it, and it
starts with a test that proves gocql behaves that way for every CQL type. When it ships, `dbmeta` can
drop its workaround.

### D11. A test that needs a server reads CQL_DSN. Decided.

Ken accepted this on 2026-09-27.

The tests today need a Cassandra on `127.0.0.1` and read `flag` values in
`TestMain`. `go test ./...` fails on any machine without one.

From here on, a test that needs a server reads a DSN in the D5 format from
`CQL_DSN`, and calls `t.Skip` when it is empty. Every other test runs with no
server. There are no build tags and no testcontainers.

A server is started with `dbrun` from `dbmeta`, and never by hand:

```bash
cd ../dbmeta/test && go run ./cmd/dbrun start cassandra && go run ./cmd/dbrun dsn cassandra
```

`dbmeta` already runs Cassandra 3.11, 4.0, 4.1 and 5.0 this way, from
`dbmeta/test/cmd/dbrun/image/cassandra.Containerfile`. How CI gets a server is D20.

### D12. Exported names do not stutter. Decided. Amended by D17.

Amended by D17: `Connector` has no exported `Config` field. `NewConnector`
takes a `*gocql.ClusterConfig` and keeps a copy in an unexported field, so
that a change the caller makes later cannot reach a running session. The
rest of the table stands. Ken accepted this entry on 2026-09-27.

`cql.CqlDriverStruct`, `cql.CqlConnector` and `cql.CqlStmt` say the package
name twice. D1 already breaks every importer, so the names change in the same
release:

| Now | After |
| --- | --- |
| `CqlDriverStruct` | `Driver` |
| `CqlConnector` | `Connector` |
| `CqlConnector.ClusterConfig` | `Connector.Config` |
| `CqlStmt` | unexported |
| `cqlConnStruct`, `cqlRowsStruct`, `cqlResultStruct` | `conn`, `rows`, `result` |
| `ErrNotImplementedYet` | removed, because nothing returns it |

`CqlStmt` is exported only to let a caller reach `CqlQuery`. The comment
beside it says that works only "if Go sql every gives access to the driver",
and `database/sql` does not. So it adds API surface and does nothing.

### D13. golangci-lint runs in CI at a pinned version. Decided.

`.golangci.yml` uses `version: "2"` and `default: all`, with a disable list
and the reason written beside each entry. The version is pinned in the
workflow, as `dburl` pins `v2.13.2`, because `default: all` means a new release
can turn on a linter nobody chose.

The rule for a finding is the one `dbmeta/CLAUDE.md` holds. A linter that
makes idiomatic Go worse is disabled, with the reason. Only a real defect gets
a code change. A change made to quiet a linter is itself a defect.

### D14. The licence stays as upstream wrote it. Decided.

`LICENSE` is MIT, copyright 2018 MichaelS11. A fork keeps its upstream licence
and notices. Any change to the licence or to a file header goes to Ken first.

### D15. Agent skills are committed as copies. Decided.

This is the rule that `dbmeta` D89 records. Ken decided it on 2026-09-27 for
every `xo` repository.

The repository carries two agent skills. A skill is a set of instructions
that a coding agent loads for a task. `simple-english` sets how prose is
written, and `go-pedantry` sets how Go is written. `skills-lock.json` names
the source of each one. The `skills` command from npm writes that file, and
`CONTRIBUTING.md` holds the command that adds or updates a skill.

Each skill is an ordinary folder in two places. Codex and the other agents
read `.agents/skills/<name>`, and Claude Code reads `.claude/skills/<name>`.
The `skills` command writes the second one as a symbolic link by default.
Git on Windows writes a symbolic link as a small text file when
`core.symlinks` is off, which is the default there. Claude Code then loads no
skill and reports nothing. So the command takes `--copy`, and both places
hold real files.

Two copies can drift apart. `TestSkillsAreCopies` in `skills_test.go` fails
on a link, on a missing copy, on two copies that differ, and on a skill
folder that `skills-lock.json` does not name.

`.gitattributes` makes every text file LF on every checkout, so that the two
copies keep the same bytes on Windows. `.gitignore` ignores
`.claude/settings.local.json`, which holds the Claude Code permissions of one
person. The shared `.claude/settings.json` is not ignored.

### D16. Every text a person reads follows simple-english. Decided.

Ken asked for this on 2026-09-27. Load the `simple-english` skill before you
write any text that a person reads: a document, a code comment, an error
message or a commit message. Follow it for that text.

Its rules include the Go conventions for error messages in D7. It adds more:
short sentences, the active voice, `can`, `will` and `must` in place of
`should` and `may`, no semicolons, no em dashes, no contractions, the
condition before the command, and one word for one meaning.

### D17. The driver is rewritten to docs/DESIGN.md. Decided. Amends D12.

Ken accepted this on 2026-09-27.

[DESIGN.md](DESIGN.md) is the target design. It came from the Go 1.27.1 and
gocql `v2.1.2` source, from Gemini and DeepSeek, and from the `n1ql` session,
on 2026-09-27. W16 builds it.

The design implements D6 to D12 and adds these choices:

- One gocql session for each `Connector`, and a `conn` with no state. This
  answers what was open question 3.
- `RowsColumnScanner`, new in Go 1.27, so that gocql decodes into the
  destination of the caller. It is also how D10 is met.
- `NamedValueChecker`, so that collections, UUIDs and user defined types can
  be bound.
- Query options as typed arguments. D23 adds the context as a second
  source.
- `driver.ResultNoRows`, `ErrNoTransactions`, and no `driver.ErrBadConn`
  from a statement.
- The file layout that `n1ql` D14 proposes, so that one driver reads like
  the other.

A NULL scanned into a plain `*string` or `*int64` becomes an error, as it is
in `lib/pq` and `go-sql-driver/mysql`. Today it is the zero value. W14 must
find every caller that depends on the old behavior.

### D18. This repository carries the modern driver. Decided.

Ken decided this on 2026-09-27. It answers what was open question 1.

`xo/cqlsql` is a second fork of the same upstream. Its working tree holds
uncommitted work on the same code: renamed files, the Go 1.10 files removed,
and the move to `apache/cassandra-gocql-driver/v2`. W17 reads that work and
brings over anything that this repository lacks. After W17, Ken archives
`xo/cqlsql`. No agent archives it.

### D19. The first tag is v0.1.0. Decided.

Ken decided this on 2026-09-27. It answers what was open question 2.

`github.com/xo/cql` has no tags. The upstream tags `v0.1.0` and `v0.1.1`
belong to `github.com/MichaelS11/go-cql-driver`, which is a different module.
Go does not compare version numbers across two module paths, so `v0.1.0`
here does not look older than `v0.1.1` there. W14 waits on this tag.

### D20. CI starts Cassandra with dbrun from a pinned dbmeta commit. Decided.

Ken decided this on 2026-09-27. It answers what was open question 4.

The workflow checks out `xo/dbmeta` at a pinned commit. Each integration job
runs `cd test && go run ./cmd/dbrun start <release>`, reads the DSN from
`dbrun dsn <release>`, and sets `CQL_DSN` to it. The jobs set
`DBMETA_RUNNER=docker` on GitHub runners.

This keeps the rule that only `dbrun` starts a server, and it gives the tests
the `dbmeta` image, which has `PasswordAuthenticator`. The cost is that
building `dbrun` builds every driver that `dbmeta` tests, and two of them
need cgo. A new pin is a commit in this repository.

The DSN that `dbrun dsn` prints must be one that this driver parses. W4 has a
test for it.

### D21. The tested releases are the ones dbmeta tests. Decided.

Ken decided this on 2026-09-27. It answers what was open question 5.

Today that is Cassandra 3.11, 4.0, 4.1 and 5.0. ScyllaDB joins when `dbmeta`
adds it, and Ken asked `dbmeta` to do that work. `dburl` sends its `scylla`
and `scy` schemes to this driver, and nothing tests them until then.

The list lives in `dbmeta`, and this repository does not repeat it. CI reads
the release names from `dbrun list --json` and keeps the Cassandra and
ScyllaDB entries, as `dbmeta` D69 does for its own matrix.

### D22. The duration helpers are removed. Decided.

Ken decided this on 2026-09-27. It answers what was open question 6.

`DurationToDuration` and `InterfaceToDuration` go in W16. Nothing in `usql`,
`dbmeta`, `dburl` or `dbtpl` calls them. Both count a month as 30.4375 days,
and neither detects an overflow. A caller scans a CQL `duration` into
`gocql.Duration`, which keeps the months, the days and the nanoseconds apart.

### D23. A query option can come from an argument or from the context. Decided.

Ken decided this on 2026-09-27. It answers what was open question 7.

Every query option type, such as `cql.PageSize`, satisfies one unexported
interface. A caller can pass an option in the argument list:

```go
rows, err := db.QueryContext(ctx, q, id, cql.PageSize(500))
```

A caller can also attach options to a context, and every call that uses the
context gets them:

```go
ctx = cql.WithOptions(ctx, cql.Consistency(gocql.LocalQuorum))
```

For each call, the driver applies the options from the context first, then
the options from the arguments. So an argument overrides the context. The
`conn` stores nothing between calls.

Gemini proposed arguments and DeepSeek proposed the context. `n1ql` has the
same question as its Q10, and it decides its own answer.

### D24. The DSN can be a URL. Decided. Amends D5.

Ken decided this on 2026-09-27.

A DSN that starts with `cassandra://` or `cql://` is a URL, and the driver
parses it with `net/url`. Any other DSN is the D5 form, which stays accepted
so that an old DSN keeps working. The driver looks at the prefix and not at
whether `url.Parse` succeeds, because `url.Parse` reads the D5 form
`h1:9042,h2?keyspace=ks` as a URL with the scheme `h1`.

The parts of the URL map to the configuration like this:

| Part | Configuration |
| --- | --- |
| user information | `username` and `password` |
| host | the hosts, separated by commas, each with an optional port |
| path | the keyspace, as one segment |
| query | the D5 keys, and a `host` key that can repeat |

`net/url` refuses a host list that holds an IPv6 address in brackets. Its
error is `invalid IP-literal`, measured with Go 1.27.1. One IPv6 address in
the host part works. So each `host` key in the query adds one host, and a
list that holds IPv6 addresses uses it:
`cql://[::1]:9042/ks?host=[::2]:9042&host=h3`.
`url.Parse` accepts a host list with commas, but `Hostname` and `Port` do not
split it, so the driver splits the host part itself.

A setting that appears in two places is an error. For example, a user name
in the user information and a `username` key is an error, and so is a
keyspace in the path and a `keyspace` key.

After the rewrite, `dburl` sends the URL itself as the DSN. `dburl` owns the
list of scheme aliases, such as `ca`, `scylla` and `datastax`, so it rewrites
the scheme to `cql://` before it sends the URL. Ken decided that on
2026-09-27. This driver accepts only `cassandra://` and `cql://`, and it never
repeats the alias list. W14 makes the change in `dburl`.

`FormatDSN` writes the URL form.

## Open questions

Do not decide an open question yourself. Ask Ken. When one is answered, it
becomes a numbered decision above and leaves this list.

There are no open questions. Ken answered the first seven on 2026-09-27, as
D17 to D23 record.
