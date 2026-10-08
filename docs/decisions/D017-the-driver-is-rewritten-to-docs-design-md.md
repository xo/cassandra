# D17. The driver is rewritten to docs/DESIGN.md

Status: Decided. Amends D12. Amended by D25 and D32.

Amended by D25: a uuid column decodes into the standard uuid.UUID, not into
a string.

Ken accepted this on 2026-09-27.

[DESIGN.md](../DESIGN.md) is the target design. It came from the Go 1.27.1 and
gocql `v2.1.2` source, from Gemini and DeepSeek, and from the `n1ql` session,
on 2026-09-27. W16 builds it.

The design implements D6 to D12 and adds these choices:

- One gocql session for each `Connector`, and a `conn` with no state. This
  answers what was open question 3.
- `RowsColumnScanner`, new in Go 1.27, so that gocql decodes into the
  destination of the caller. It is also how D10 is met.
- `NamedValueChecker`, so that collections, UUIDs and user defined types can
  be bound.
- Query options as typed arguments. D23 adds the context as a second
  source.
- `driver.ResultNoRows`, `ErrNoTransactions`, and no `driver.ErrBadConn`
  from a statement.
- The file layout that `n1ql` D14 proposes, so that one driver reads like
  the other.

A NULL scanned into a plain `*string` or `*int64` becomes an error, as it is
in `lib/pq` and `go-sql-driver/mysql`. Today it is the zero value. W14 must
find every caller that depends on the old behavior.
