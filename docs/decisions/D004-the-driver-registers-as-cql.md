# D4. The driver registers as cql

Status: Replaced by D27.

`sql.Register("cql", ...)` stays. `dburl` maps its `cql` scheme and the aliases
`ca`, `cassandra`, `datastax`, `scy` and `scylla` to the driver name `cql`
(`dburl/scheme.go`), and
`usql` and `dbmeta` open it by that name. A new name breaks all
three and gains nothing.
