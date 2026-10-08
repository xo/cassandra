# D3. The repository uses the xo layout

Status: Decided. Amended by D27, D28, D29 and D30.

This matches `dbmeta` and `usql`:

- The root holds `README.md`, `CLAUDE.md` and `CONTRIBUTING.md`, and no other
  Markdown. Every other document is in `docs/`.
- `CLAUDE.md` follows the order `dbmeta/CLAUDE.md` uses: purpose, a "Which
  document to read" table, numbered hard rules with their reasons, layout, Go
  conventions, linting, the commands to run before committing, and how to
  write documentation.
- `CONTRIBUTING.md` holds the same material for a person, and it is shorter.
- `docs/PLAN.md` is this file. It holds decisions only.
- `docs/BACKLOG.md` holds work items, in the format `usql/docs/BACKLOG.md`
  uses.
- The package stays flat, with one root package named `cql`. A sub-package is
  added only where there is a real boundary.
- Examples are `Example` tests. Golden files go in `testdata/`.
- CI is one GitHub Actions workflow, `.github/workflows/test.yml`, on
  `ubuntu-latest`, with `go-version-file: go.mod`. There is no Makefile.

`dbmeta` has a test, `docs_test.go`, that fails on a stray Markdown file at the
root or a document missing from the table in `CLAUDE.md`. This repository gets
the same test, in W1.
