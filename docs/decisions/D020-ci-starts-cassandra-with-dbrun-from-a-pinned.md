# D20. CI starts Cassandra with dbrun from a pinned dbmeta commit

Status: Decided.

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
