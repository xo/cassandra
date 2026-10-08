# D15. Agent skills are committed as copies

Status: Decided.

This is the rule that `dbmeta` D89 records. Ken decided it on 2026-09-27 for
every `xo` repository.

The repository carries two agent skills. A skill is a set of instructions
that a coding agent loads for a task. `simple-english` sets how prose is
written, and `go-pedantry` sets how Go is written. `skills-lock.json` names
the source of each one. The `skills` command from npm writes that file, and
`CONTRIBUTING.md` holds the command that adds or updates a skill.

Each skill is an ordinary folder in two places. Codex and the other agents
read `.agents/skills/<name>`, and Claude Code reads `.claude/skills/<name>`.
The `skills` command writes the second one as a symbolic link by default.
Git on Windows writes a symbolic link as a small text file when
`core.symlinks` is off, which is the default there. Claude Code then loads no
skill and reports nothing. So the command takes `--copy`, and both places
hold real files.

Two copies can drift apart. `TestSkillsAreCopies` in `skills_test.go` fails
on a link, on a missing copy, on two copies that differ, and on a skill
folder that `skills-lock.json` does not name.

`.gitattributes` makes every text file LF on every checkout, so that the two
copies keep the same bytes on Windows. `.gitignore` ignores
`.claude/settings.local.json`, which holds the Claude Code permissions of one
person. The shared `.claude/settings.json` is not ignored.
