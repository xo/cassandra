# D1. The module path is github.com/xo/cql

Status: Replaced by D27.

`go.mod` still says `github.com/MichaelS11/go-cql-driver`. Upstream is
archived on GitHub and last took a change on 2020-09-20. Every `xo` module is
named `github.com/xo/<repository>`.

The new path is a breaking change for every importer, so it is the one chance
to make the other breaking changes in this file. `usql`, `dbmeta` and `dburl`
all name the old path, and W14 moves them.
