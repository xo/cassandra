# D25. A uuid column is the standard uuid.UUID

Status: Decided. Amends D17.

Ken decided this on 2026-09-27.

Go 1.27 has a `uuid` package in the standard library, with
`type UUID [16]byte`. gocql knows only `gocql.UUID`: its marshal and unmarshal
code switch on `gocql.UUID` and `[16]byte`, and a named type such as
`uuid.UUID` matches neither. `uuid.UUID` has no `Value` or `Scan` method. So
before this decision the driver refused to bind a `uuid.UUID`, and refused
to scan into one, as a test showed.

Now the driver handles the standard type itself:

- `CheckNamedValue` converts a `uuid.UUID`, a `*uuid.UUID` and a
  `sql.Null[uuid.UUID]` to a `gocql.UUID` before gocql sees it. Both types
  are a `[16]byte`, so the conversion copies nothing.
- The canonical value of a `uuid` or a `timeuuid` column is a `uuid.UUID`.
  `*any`, a `sql.Scanner`, `Next` and `ColumnTypeScanType` all get it. D17
  set a string, and this decision amends that.
- `ScanColumn` sends a `*uuid.UUID` and a `**uuid.UUID` to
  `sql.ConvertAssign` with the canonical value. `sql.Null[uuid.UUID]` works
  with no special case, because the canonical value is assignable to its
  field.

A `*gocql.UUID` and a `*string` still go to gocql, as before.

The cost is that `sql.NullString` and any other `sql.Scanner` that accepts
only a string can no longer read a `uuid` column. Scan into `*string` for the
text form.

A collection that holds the standard type, such as `[]uuid.UUID` for a
`list<uuid>`, is not supported in either direction. gocql does not convert
the elements, and a conversion by reflection is much more code for a rare
case. Ken chose to leave it out.
