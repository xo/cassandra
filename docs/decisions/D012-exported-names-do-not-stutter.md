# D12. Exported names do not stutter

Status: Decided. Amended by D17.

Amended by D17: `Connector` has no exported `Config` field. `NewConnector`
takes a `*gocql.ClusterConfig` and keeps a copy in an unexported field, so
that a change the caller makes later cannot reach a running session. The
rest of the table stands. Ken accepted this entry on 2026-09-27.

`cql.CqlDriverStruct`, `cql.CqlConnector` and `cql.CqlStmt` say the package
name twice. D1 already breaks every importer, so the names change in the same
release:

| Now | After |
| --- | --- |
| `CqlDriverStruct` | `Driver` |
| `CqlConnector` | `Connector` |
| `CqlConnector.ClusterConfig` | `Connector.Config` |
| `CqlStmt` | unexported |
| `cqlConnStruct`, `cqlRowsStruct`, `cqlResultStruct` | `conn`, `rows`, `result` |
| `ErrNotImplementedYet` | removed, because nothing returns it |

`CqlStmt` is exported only to let a caller reach `CqlQuery`. The comment
beside it says that works only "if Go sql every gives access to the driver",
and `database/sql` does not. So it adds API surface and does nothing.
