# D21. The tested releases are the ones dbmeta tests

Status: Decided.

Ken decided this on 2026-09-27. It answers what was open question 5.

Today that is Cassandra 3.11, 4.0, 4.1 and 5.0. ScyllaDB joins when `dbmeta`
adds it, and Ken asked `dbmeta` to do that work. `dburl` sends its `scylla`
and `scy` schemes to this driver, and nothing tests them until then.

The list lives in `dbmeta`, and this repository does not repeat it. CI reads
the release names from `dbrun list --json` and keeps the Cassandra and
ScyllaDB entries, as `dbmeta` D69 does for its own matrix.
