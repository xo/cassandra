# D8. The package holds no configuration

Status: Decided.

Ken accepted this on 2026-09-27.

`CqlDriver` is an exported package variable with a mutable `Logger`, and it
logs to standard error by default. `DbConsistencyLevels` and `DbConsistency`
are exported maps that any importer can change. `dbmeta` hard rule 6 says why
this matters: one driver in `usql` borrowed another's configuration and
shipped a fault.

So the registered driver value is unexported. The consistency tables are
unexported. The driver logs nothing, and every fact it used to log goes into
the error it returns instead, which is D9. If a caller needs gocql's own log,
it sets `Logger` on the `ClusterConfig` it passes to the `Connector`. That
field exists in `v2` (D6).

`usql` writes `cql.CqlDriver.Logger` today. W14 removes that line.
