# D9. An error always reaches the caller

Status: Decided. Amended by D33.

Two places lose an error today.

`Ping` replaces every failure with `driver.ErrBadConn`, and logs the cause to
standard error. A wrong password therefore arrives as "driver: bad
connection". `database/sql` retries on `ErrBadConn`, so a failure that cannot
succeed on retry is tried again. From here on, `ErrBadConn` is returned only
when a retry on a new connection can help, and it wraps the cause.
`database/sql` in Go 1.27 matches it with `errors.Is`, so a wrapped
`ErrBadConn` still triggers the retry.

`QueryContext` never returns the error from the statement. It builds the
iterator, reads its columns, and returns rows. The error arrives only once a
caller iterates. `usql` #58 and the Cassandra work in `dbmeta` (D62, "cost an
afternoon") both saw an empty result where the server had refused the
statement. From here on, `QueryContext` returns the server's error itself.
