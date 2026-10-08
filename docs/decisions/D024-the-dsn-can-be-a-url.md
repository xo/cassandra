# D24. The DSN can be a URL

Status: Decided. Amends D5. Amended by D27, D31 and D34.

Ken decided this on 2026-09-27.

A DSN that starts with `cassandra://` or `cql://` is a URL, and the driver
parses it with `net/url`. Any other DSN is the D5 form, which stays accepted
so that an old DSN keeps working. The driver looks at the prefix and not at
whether `url.Parse` succeeds, because `url.Parse` reads the D5 form
`h1:9042,h2?keyspace=ks` as a URL with the scheme `h1`.

The parts of the URL map to the configuration like this:

| Part | Configuration |
| --- | --- |
| user information | `username` and `password` |
| host | the hosts, separated by commas, each with an optional port |
| path | the keyspace, as one segment |
| query | the D5 keys, and a `host` key that can repeat |

`net/url` refuses a host list that holds an IPv6 address in brackets. Its
error is `invalid IP-literal`, measured with Go 1.27.1. One IPv6 address in
the host part works. So each `host` key in the query adds one host, and a
list that holds IPv6 addresses uses it:
`cql://[::1]:9042/ks?host=[::2]:9042&host=h3`.
`url.Parse` accepts a host list with commas, but `Hostname` and `Port` do not
split it, so the driver splits the host part itself.

A setting that appears in two places is an error. For example, a user name
in the user information and a `username` key is an error, and so is a
keyspace in the path and a `keyspace` key.

After the rewrite, `dburl` sends the URL itself as the DSN. `dburl` owns the
list of scheme aliases, such as `ca`, `scylla` and `datastax`, so it rewrites
the scheme to `cql://` before it sends the URL. Ken decided that on
2026-09-27. This driver accepts only `cassandra://` and `cql://`, and it never
repeats the alias list. W14 makes the change in `dburl`.

`FormatDSN` writes the URL form.
