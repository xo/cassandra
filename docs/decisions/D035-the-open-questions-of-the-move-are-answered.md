# D35. The open questions of the move are answered

Status: Decided. Amends D32.

Ken answered the open questions of the move to `dbimp` on 2026-10-08. D34 holds
the answer about the list of hosts.

- The first tag is `v0.1.0`. Ken tags the module, and nothing here tags it.
- `github.com/xo/cql` gets nothing: no notice and no final tag.
- This repository is the Cassandra driver. `dbimp` does not write one, and the
  earlier plan that `xo/cql` gets no more work once `dbimp` has a Cassandra driver
  does not stand.
- The dialect constant `dbmeta.Cassandra` becomes `"cassandra"`, so that the name
  `cql` goes from `dbmeta` too. The change is in `dbmeta`.
- A varint is the kind `big integer`, with the Go type `*big.Int`, and the kind
  means an integer of any size above `int64`. The words of the types document of
  `dbimp` name 128 and 256 bits, so they change in `dbimp`.
- A map whose key is not text stays the map that gocql chooses, such as
  `map[int]string`, and CASSANDRA.md says so.
- The driver keeps its own `contract_test.go`, and `dbimp` gets no row for it.
- `dbmeta` is asked for an ordinary Cassandra role, as it has one for Couchbase,
  so that the integration tests run as both principals.
- `dburl` names the scheme and the driver `cassandra`, keeps `cql`, `ca`,
  `datastax`, `scy` and `scylla` as aliases, writes the URL `cassandra://`, and
  sets the `GoPackage` to `github.com/xo/cassandra`. `dbmeta` and `usql` follow.
