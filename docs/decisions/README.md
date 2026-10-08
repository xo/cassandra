# Decisions

Every decision of this repository is a file in this folder, one for each
decision, named by its number and its title. This table is the index. Find
the number here, then open the file.

Each file opens with its status. `Decided` means that Ken settled it, or that
it follows from a rule every `xo` repository keeps. `Proposed` means a
recommendation that waits for Ken. A decision that changes an earlier one
says so in its status, as "Amends D5", and the earlier one says it back, as
"Amended by D24". Read the status before the decision.

A decision is never edited to change its conclusion. A later decision that
replaces or amends one names it, and the older one gains a status that names
the later one. A new decision gets the next number and a file of its own.
Add its row here. `TestTheDecisionIndexIsComplete` fails when a decision has
no row or a row is wrong. `TestAnAmendmentPointsBothWays` fails when an
amendment names only one side. D29 moved the decisions here from
`docs/PLAN.md`. Work items are in [BACKLOG.md](../BACKLOG.md), and the two
series never mix.

| # | Decision | Status |
| --- | --- | --- |
| [D1](D001-the-module-path-is-github-com-xo-cql.md) | The module path is github.com/xo/cql | Replaced by D27 |
| [D2](D002-the-go-directive-is-1-27-1.md) | The go directive is 1.27.1 | Decided |
| [D3](D003-the-repository-uses-the-xo-layout.md) | The repository uses the xo layout | Decided. Amended by D27, D28, D29 and D30 |
| [D4](D004-the-driver-registers-as-cql.md) | The driver registers as cql | Replaced by D27 |
| [D5](D005-the-dsn-is-the-one-dburl-writes.md) | The DSN is the one dburl writes | Replaced by D31. Amended by D24 |
| [D6](D006-the-driver-moves-to-apache-cassandra-gocql-driver.md) | The driver moves to apache/cassandra-gocql-driver/v2 | Decided |
| [D7](D007-go-conventions-are-the-ones-dburl-and-dbmeta-keep.md) | Go conventions are the ones dburl and dbmeta keep | Decided |
| [D8](D008-the-package-holds-no-configuration.md) | The package holds no configuration | Decided |
| [D9](D009-an-error-always-reaches-the-caller.md) | An error always reaches the caller | Decided. Amended by D33 |
| [D10](D010-a-cql-null-reaches-database-sql-as-nil.md) | A CQL NULL reaches database/sql as nil | Decided |
| [D11](D011-a-test-that-needs-a-server-reads-cql-dsn.md) | A test that needs a server reads CQL_DSN | Decided. Amended by D27 |
| [D12](D012-exported-names-do-not-stutter.md) | Exported names do not stutter | Decided. Amended by D17 |
| [D13](D013-golangci-lint-runs-in-ci-at-a-pinned-version.md) | golangci-lint runs in CI at a pinned version | Decided |
| [D14](D014-the-licence-stays-as-upstream-wrote-it.md) | The licence stays as upstream wrote it | Decided |
| [D15](D015-agent-skills-are-committed-as-copies.md) | Agent skills are committed as copies | Decided |
| [D16](D016-every-text-a-person-reads-follows-simple-english.md) | Every text a person reads follows simple-english | Decided |
| [D17](D017-the-driver-is-rewritten-to-docs-design-md.md) | The driver is rewritten to docs/DESIGN.md | Decided. Amends D12. Amended by D25 and D32 |
| [D18](D018-this-repository-carries-the-modern-driver.md) | This repository carries the modern driver | Decided |
| [D19](D019-the-first-tag-is-v0-1-0.md) | The first tag is v0.1.0 | Decided |
| [D20](D020-ci-starts-cassandra-with-dbrun-from-a-pinned.md) | CI starts Cassandra with dbrun from a pinned dbmeta commit | Decided |
| [D21](D021-the-tested-releases-are-the-ones-dbmeta-tests.md) | The tested releases are the ones dbmeta tests | Decided |
| [D22](D022-the-duration-helpers-are-removed.md) | The duration helpers are removed | Decided |
| [D23](D023-a-query-option-can-come-from-an-argument-or-from.md) | A query option can come from an argument or from the context | Decided. Amended by D33 |
| [D24](D024-the-dsn-can-be-a-url.md) | The DSN can be a URL | Decided. Amends D5. Amended by D27, D31 and D34 |
| [D25](D025-a-uuid-column-is-the-standard-uuid-uuid.md) | A uuid column is the standard uuid.UUID | Decided. Amends D17 |
| [D26](D026-the-decimal-type-stays-gopkg-in-inf-v0.md) | The decimal type stays gopkg.in/inf.v0 | Replaced by D32. Amended by D30 |
| [D27](D027-the-module-the-package-and-the-driver-are-named.md) | The module, the package and the driver are named cassandra | Decided. Replaces D1 and D4. Amends D3, D11 and D24. Amended by D31 |
| [D28](D028-the-repository-sets-up-agents-as-d110-of-dbmeta.md) | The repository sets up agents as D110 of dbmeta says | Decided. Amends D3 |
| [D29](D029-the-decisions-are-one-file-each.md) | The decisions are one file each | Decided. Amends D3 |
| [D30](D030-the-driver-follows-the-drivers-of-dbimp.md) | The driver follows the drivers of dbimp | Decided. Amends D3 and D26 |
| [D31](D031-a-dsn-is-a-url-whose-scheme-is-cassandra.md) | A DSN is a URL whose scheme is cassandra | Decided. Replaces D5. Amends D24 and D27. Amended by D34 |
| [D32](D032-the-types-are-the-kinds-of-dbimp.md) | The types are the kinds of dbimp | Decided. Amends D17. Replaces D26. Amended by D35 |
| [D33](D033-the-options-and-the-errors-are-those-of-dbimp.md) | The options and the errors are those of dbimp | Decided. Amends D9 and D23 |
| [D34](D034-the-host-part-of-a-dsn-holds-one-host.md) | The host part of a DSN holds one host | Decided. Amends D24 and D31 |
| [D35](D035-the-open-questions-of-the-move-are-answered.md) | The open questions of the move are answered | Decided. Amends D32 |
