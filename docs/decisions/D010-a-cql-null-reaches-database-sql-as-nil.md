# D10. A CQL NULL reaches database/sql as nil

Status: Decided.

gocql decodes a NULL of any type as the zero value of that type, and
`interfaceToValue` passes the zero value on. So `SELECT (text)NULL` scanned
into `sql.Null[string]` comes back `Valid` and `""`. `dbmeta` measured this,
and it works around it with a scanner that discards a padded column
(`dbmeta/docs/PLAN.md` D62, and the Cassandra section of
`dbmeta/docs/NULLS.md`). A real catalog column that is NULL cannot be rescued
there at all.

The driver fixes this by scanning each column into a pointer to a pointer,
so that gocql can leave the inner pointer nil for a NULL. W7 does it, and it
starts with a test that proves gocql behaves that way for every CQL type. When it ships, `dbmeta` can
drop its workaround.
