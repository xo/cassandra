# D5. The DSN is the one dburl writes

Status: Replaced by D31. Amended by D24.

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
