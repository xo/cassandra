# D29. The decisions are one file each

Status: Decided. Amends D3.

Ken decided on 2026-10-08 that this repository keeps each decision in a file
of its own, as `dbmeta` D111 and `dbimp` D72 describe. `docs/PLAN.md` held
D1 to D28 and the open questions in one file, which D3 decided, so this
amends D3.

Each decision is `docs/decisions/D<nnn>-<title>.md`, with the number in three
digits, so that a listing sorts in order. The file opens with its number and
title, a blank line, and its status, such as `Status: Decided.`
`docs/decisions/README.md` is the index. `docs/PLAN.md` keeps the purpose of
the project, what exists, the testing plan and the open questions for Ken.

A reference by number, such as D5, still works, because it names the number
and not a place in a file. The move kept the number and the text of every
decision. It gave D1, D3, D4, D11 and D24 a status that names D27 or D28, and
the line that said so inside the text went.
