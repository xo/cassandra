# D31. A DSN is a URL whose scheme is cassandra

Status: Decided. Replaces D5. Amends D24 and D27. Amended by D34.

Ken decided on 2026-10-08 that the DSN is a standard URL whose scheme is the
one name of the driver, and that the driver keeps no alias and no older form
(dbimp D27, dbimp D28 and dbimp D35). `dburl` turns every alias, such as `ca`,
`scylla` and `cql`, into this URL.

- `ParseDSN` reads `cassandra://` and nothing else. `cql://`, a bare list of
  hosts with a query, and any other scheme are errors. D27 had kept `cql://` as
  an alias for a short time, and D5 had kept the list of hosts.
- It parses with `net/url`. The errors wrap `dbimp.ErrScheme`,
  `dbimp.ErrUnknownKey`, `dbimp.ErrRepeatedKey` and `dbimp.ErrInvalidValue`, and
  never hold the DSN, which can hold a password.
- Every key has the default of `gocql.NewCluster`. A key that the driver does not
  know is an error.
- `FormatDSN` writes the URL that `ParseDSN` reads back.
- A list of hosts is the one thing that a standard URL does not hold. The driver
  keeps what D24 made, a list in the host part and a repeated `host` key, until
  Ken decides how a list is written (open question 13).
