# D19. The first tag is v0.1.0

Status: Decided.

Ken decided this on 2026-09-27. It answers what was open question 2.

`github.com/xo/cql` has no tags. The upstream tags `v0.1.0` and `v0.1.1`
belong to `github.com/MichaelS11/go-cql-driver`, which is a different module.
Go does not compare version numbers across two module paths, so `v0.1.0`
here does not look older than `v0.1.1` there. W14 waits on this tag.
