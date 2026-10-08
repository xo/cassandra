# Progress

This file says where the work stands, so that a session that ends can resume
from it. Update it at the end of each step.

## Done in the working tree, and not committed

- The rename to `cassandra` (D27, W20). The module, the package, the driver
  name and the test variable `CASSANDRA_DSN` changed.
- `AGENTS.md` holds the rules, with the standing rules first, and `CLAUDE.md`
  imports it (D28). The decisions are one file each in `docs/decisions/`
  (D29).
- The move to the drivers of `dbimp` (D30 to D33, W21):
  - The DSN is `cassandra://` only, read with `dbimp.NewQuery` (D31).
  - The types are the kinds of `dbimp` (D32). `docs/CASSANDRA.md` holds the type
    table and the interface table, and `tables_test.go` writes them.
  - The options and the errors are those of `dbimp` (D33).
  - `contract_test.go`, `options_test.go`, `types_test.go` and
    `roundtrip_integration_test.go` are new.
  - `.golangci.yml` and the workflow follow `dbimp`.
- On 2026-10-08 the integration tests passed on `cassandra-3.11`,
  `cassandra-5.0`, `scylla-2025.1` and `scylla-2026.3`, started with `dbrun`
  and removed after the run. `gofmt`, `go vet`, `go build`, `go test -race
  -count=2` and `golangci-lint` report nothing.

## Next

1. Stage the work. The session did not run `git add`, because the tool refused
   it, so Ken must stage it or allow the command.
2. The peers `dburl`, `usql` and `dbmeta` move by their own rules (W20 and W21).
   The message to `usql` went out in the first pass. The messages to `dburl`
   and `dbmeta` did not, so Ken must tell them. `dburl` disagrees with the DSN
   (D35).

3. `dbmeta` will make the `dsn` field of every Cassandra and ScyllaDB release the
   URL `cassandra://...`, and add the ordinary role `dbmeta_user` (its reply of
   2026-10-08). When its commit exists, move `DBMETA_COMMIT` in the workflow to
   it, read `.[0].dsn` instead of `.[0].url`, drop the `sed`, and run the tests
   as the ordinary principal too. `dbmeta` and `dburl` move to the new name only
   when the module has a tag, so tell both sessions the tag when Ken tags it.

## Open for Ken

Ken answered the open questions on 2026-10-08 (D34 and D35). Two remain, and both
are work in other repositories: an ordinary Cassandra role in `dbmeta` (open
question 18), and the scheme and the driver `cassandra` in `dburl` (open question
19). `usql` and `dbmeta` follow `dburl` (W21). Ken tags `v0.1.0` when he says.
