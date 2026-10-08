# D13. golangci-lint runs in CI at a pinned version

Status: Decided.

`.golangci.yml` uses `version: "2"` and `default: all`, with a disable list
and the reason written beside each entry. The version is pinned in the
workflow, as `dburl` pins `v2.13.2`, because `default: all` means a new release
can turn on a linter nobody chose.

The rule for a finding is the one `dbmeta/CLAUDE.md` holds. A linter that
makes idiomatic Go worse is disabled, with the reason. Only a real defect gets
a code change. A change made to quiet a linter is itself a defect.
