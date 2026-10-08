# D11. A test that needs a server reads CQL_DSN

Status: Decided. Amended by D27.

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
