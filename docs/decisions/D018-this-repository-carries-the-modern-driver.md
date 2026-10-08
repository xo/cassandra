# D18. This repository carries the modern driver

Status: Decided.

Ken decided this on 2026-09-27. It answers what was open question 1.

`xo/cqlsql` is a second fork of the same upstream. Its working tree holds
uncommitted work on the same code: renamed files, the Go 1.10 files removed,
and the move to `apache/cassandra-gocql-driver/v2`. W17 reads that work and
brings over anything that this repository lacks. After W17, Ken archives
`xo/cqlsql`. No agent archives it.
