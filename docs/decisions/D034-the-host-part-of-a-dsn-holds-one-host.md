# D34. The host part of a DSN holds one host

Status: Decided. Amends D24 and D31.

Ken decided this on 2026-10-08, and it answers open question 13. A standard URL
holds one host, so the host part of the DSN holds one host, and each further host
is a `host` key of the query:

```text
cassandra://user:password@h1:9042/keyspace?host=h2:9042&host=[::1]:9042
```

- `ParseDSN` reads the host part as one host. A host part that holds a comma is an
  error that wraps `dbimp.ErrInvalidValue`, and net/url refuses the other forms
  of a list.
- The hosts are the host of the host part, then each `host` key in order. A DSN
  with no host connects to `127.0.0.1`.
- `FormatDSN` writes the first host in the host part when net/url reads it back
  as it is, and every other host in a `host` key, in order.
- `dbimp.NewQuery` refuses a repeated key, so the driver takes the key `host` out
  of the query before it reads the rest. This is the one key that can repeat.
- D24 had a list in the host part, separated by commas, and D31 kept it.
