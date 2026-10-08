# Plan

`cassandra` is a `database/sql` driver for Apache Cassandra and ScyllaDB. It
wraps the Apache Cassandra Go driver, which these documents call gocql (D6),
and it registers as `cassandra` (D27). The module is `github.com/xo/cassandra`.
`usql` and `dbmeta` use it, and `dburl` writes its DSN (D24).

## Architecture

The package is flat, with one root package named `cassandra` (D3). One
`Connector` owns one gocql session, and every connection of a `sql.DB` shares
it (D17). Only `session.go` calls gocql to run a statement, so the unit tests
replace gocql with a fake. [DESIGN.md](DESIGN.md) is the design that the code
follows, and every file is named in the layout table in [AGENTS.md](../AGENTS.md).

## What exists

The driver is written to [DESIGN.md](DESIGN.md) (D17, W16). It has the
`cassandra` registration, a DSN in two forms (D5, D24), query options (D23),
canonical column values (D17, D25) and the errors of D9. The module has no tag
yet. Every decision is a file in [decisions/](decisions/README.md), and the
index is there (D29). Every work item is in [BACKLOG.md](BACKLOG.md). Where the
work stands is in [PROGRESS.md](PROGRESS.md).

## Testing plan

A unit test needs no server (D11). It uses the fake in `fake_test.go`. A test
that needs a server reads `CASSANDRA_DSN` and skips when it is empty. CI starts
every Cassandra and ScyllaDB release that `dbmeta` tests, with `dbrun` from a
pinned `dbmeta` commit (D20, D21), and runs the integration tests against each.
`golangci-lint` runs at a pinned version (D13). `docs_test.go` and
`prose_test.go` check the documents, and `skills_test.go` checks the agent
skills (D15, D16).

## Open questions

Do not decide an open question yourself. Ask Ken. When one is answered, it
becomes a numbered decision in [decisions/](decisions/README.md) and leaves
this list.

Ken answered the first seven on 2026-09-27, as D17 to D23 record. The next
question takes the next number, so that "open question 1" in an older entry
still means the question that D18 answered.

18. Does `dbmeta` give Cassandra an ordinary role? Ken asked for one (D35).
    `dbrun` names only the administrator `cassandra`, so the tests run as that
    user only. The work is in `dbmeta`.
19. Does `dburl` reach this driver? Ken decided how (D35): the scheme and the
    driver `cassandra`, the aliases `cql`, `ca`, `datastax`, `scy` and `scylla`,
    the URL `cassandra://` and the `GoPackage` `github.com/xo/cassandra`. The
    work is in `dburl`, and `dbmeta` prints `scylla://` for ScyllaDB, which
    this driver refuses, until it follows (W21).
