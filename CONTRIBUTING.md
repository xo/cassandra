# Contributing to cql

`cql` is a `database/sql` driver for Apache Cassandra. Read two documents
before you change anything:

- [`docs/PLAN.md`](docs/PLAN.md) holds every decision and the open questions.
  Do not decide an open question yourself. Ask Ken.
- [`docs/BACKLOG.md`](docs/BACKLOG.md) holds the planned work, in order.

[`docs/DESIGN.md`](docs/DESIGN.md) is the design that the code follows, and
[`CLAUDE.md`](CLAUDE.md) holds the rules. `CLAUDE.md` is written for a coding
agent, and everything in it applies to a person.

## Before you send a change

Run these in the repository root. `gofmt -l .` must print nothing.

```bash
gofmt -l . && go vet ./... && go test -race -count=2 ./...
golangci-lint run ./...
```

The unit tests need no server. The integration tests run against the server
that `CQL_DSN` names, and they skip when it is empty. Start the server with
`dbrun` from [`dbmeta`](https://github.com/xo/dbmeta), in a checkout next to
this one. Do not start a container by hand (D20).

```bash
(cd ../dbmeta/test && go run ./cmd/dbrun start cassandra-5.0)
export CQL_DSN=$(cd ../dbmeta/test && go run ./cmd/dbrun dsn --json cassandra-5.0 | jq -r '.[0].dsn')
go test -race -count=1 -run Integration ./...
```

## Writing text

Load the `simple-english` skill before you write any text that a person
reads: a document, a code comment, an error message or a commit message.
Follow it for that text. See D16 in `docs/PLAN.md`.

## Agent skills

The repository carries two agent skills. A skill is a set of instructions
that a coding agent loads for a task. `simple-english` sets how prose is
written, and `go-pedantry` sets how Go is written.

`skills-lock.json` names the source of each skill. The `skills` command from
npm writes that file. Version 1.7.0 is the version that `dbmeta` measured.
The command writes each skill into two folders. Codex and the other agents
read `.agents/skills/<name>`, and Claude Code reads `.claude/skills/<name>`.

To add or update a skill, run this command in the repository root. The
example updates `simple-english`. `skills-lock.json` holds the source for
each skill:

```bash
npx skills@1.7.0 add AminBlg/SimpleEnglish --skill simple-english --agent codex claude-code --copy -y
```

Keep `--copy`. Without it, the command writes `.claude/skills/<name>` as a
symbolic link. A Windows checkout writes a symbolic link as a text file, and
Claude Code then loads no skill and reports nothing. `TestSkillsAreCopies`
fails on a link, and it fails when the two folders differ. See D15 in
`docs/PLAN.md`.

`.claude/settings.local.json` holds the Claude Code permissions of one
person. The root `.gitignore` ignores it.
