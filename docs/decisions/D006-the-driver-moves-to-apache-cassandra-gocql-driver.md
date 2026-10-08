# D6. The driver moves to apache/cassandra-gocql-driver/v2

Status: Decided.

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
