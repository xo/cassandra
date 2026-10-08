# D28. The repository sets up agents as D110 of dbmeta says

Status: Decided. Amends D3.

Ken decided this on 2026-10-08.

D110 of `dbmeta` is the standard for every `xo` repository. This repository
follows it:

- `AGENTS.md` holds the rules, and it opens with the three standing rules.
  `CLAUDE.md` holds one line, `@AGENTS.md`, and it is a file and not a link.
- The root holds four documents: `AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`
  and `README.md`. D3 said three.
- `docs_test.go` checks that `CLAUDE.md` imports `AGENTS.md`, and it reads the
  table of documents from `AGENTS.md`.
