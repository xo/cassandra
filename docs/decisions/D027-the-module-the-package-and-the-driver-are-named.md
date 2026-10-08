# D27. The module, the package and the driver are named cassandra

Status: Decided. Replaces D1 and D4. Amends D3, D11 and D24. Amended by D31.

Ken decided this on 2026-10-08. Amends D3, D11 and D24.

The name `cql` goes. CQL is the query language, and the product is Cassandra,
which `dburl`, `usql` and `dbmeta` already call it. The repository is
`github.com/xo/cassandra`, so the name of the module, the package and the
driver is `cassandra` and no longer `cql`:

- The module path is `github.com/xo/cassandra`, and the package is
  `cassandra`.
- The driver registers as `cassandra`, and a caller opens it with
  `sql.Open("cassandra", dsn)`.
- The variable that the integration tests read is `CASSANDRA_DSN`. The old
  name `CQL_DSN` does not work. Only the tests read the variable, so no caller
  depends on it.
- `FormatDSN` writes `cassandra://`. `ParseDSN` reads `cassandra://` and it
  also reads `cql://`, because a caller can hold a `cql://` DSN that an
  earlier release or a tool wrote, and refusing it gains nothing. This keeps
  the form of D5 and of D24 working. The driver does not read any other scheme
  alias. `dburl` owns that list.

This is a breaking change for the importers of `github.com/xo/cql`, which had
no tag (D19). The three peers move in the same change: `dburl` names the driver
`cassandra` and sends the URL with the scheme `cassandra://`, `usql` imports
the new path and registers `cassandra`, and `dbmeta` uses the new path and the
driver name in its tests. The CQL language keeps its name where the text means
the language: `cqlsh`, the `UnmarshalCQL` method of gocql and the lexer name.
