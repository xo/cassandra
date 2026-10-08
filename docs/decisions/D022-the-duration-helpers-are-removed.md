# D22. The duration helpers are removed

Status: Decided.

Ken decided this on 2026-09-27. It answers what was open question 6.

`DurationToDuration` and `InterfaceToDuration` go in W16. Nothing in `usql`,
`dbmeta`, `dburl` or `dbtpl` calls them. Both count a month as 30.4375 days,
and neither detects an overflow. A caller scans a CQL `duration` into
`gocql.Duration`, which keeps the months, the days and the nanoseconds apart.
