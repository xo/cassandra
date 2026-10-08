# D33. The options and the errors are those of dbimp

Status: Decided. Amends D9 and D23.

Ken decided on 2026-10-08 that the driver takes the options of one statement
through the machinery of `dbimp` (dbimp D109), and reports its errors with the
sentinel errors of `dbimp`.

- `Option` is `dbimp.Option[options]`. An option comes from the context through
  `WithOptions`, then from an argument, and a later one wins. `CheckNamedValue`
  keeps an option, and each statement calls `dbimp.Resolve`.
- The four options of every driver exist: `WithTimeout`, `WithReadonly`,
  `WithParameter` and `WithDatabase`. Cassandra has no read-only statement and no
  body of keys, so `WithReadonly(true)` and `WithParameter` fail with
  `dbimp.ErrNotSupported`. A negative timeout or page size fails with
  `dbimp.ErrInvalidValue`. `WithDatabase` sets the keyspace with
  `Query.SetKeyspace`, which needs protocol version 5.
- The options of D23 became functions: `WithConsistency`,
  `WithSerialConsistency`, `WithPageSize`, `WithIdempotent` and `WithTimestamp`.
- `BeginTx`, a named argument, an argument that gocql cannot bind and a call with
  no context fail with an error that wraps `dbimp.ErrNotSupported`. The sentinel
  errors `ErrNoTransactions`, `ErrNamedArgs`, `ErrUnsupportedArg`, `ErrNoContext`
  and `ErrInvalidDSN` are gone. `ErrConnectorClosed` stays.
- A result that fails after a row reached the caller wraps `dbimp.ErrIncomplete`
  (dbimp D21 and dbimp D107).
- The driver still never returns `driver.ErrBadConn` from a statement (D9),
  which agrees with dbimp D8.
