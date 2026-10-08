# D2. The go directive is 1.27.1

Status: Decided.

`go.mod` says `go 1.27.1` and has no `toolchain` line. That is Ken's target
and what the other `xo` repositories use. The line changed first, together
with `go mod tidy`, because `skills_test.go` needs generics (D15). The `// +build go1.10` constraints
and the files split out for them go, because no supported Go needs them.
