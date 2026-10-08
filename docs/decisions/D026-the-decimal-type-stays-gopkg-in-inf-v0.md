# D26. The decimal type stays gopkg.in/inf.v0

Status: Replaced by D32. Amended by D30.

Ken decided this on 2026-09-27.

gocql `v2.1.2` decodes a CQL `decimal` into `*inf.Dec` and binds a decimal
only from `inf.Dec`. So `gopkg.in/inf.v0` is in the module graph because
gocql requires it, whatever this driver does. The driver imports it in
`types.go` to turn a decimal into its canonical string, and the tests import
it to bind a decimal.

`inf.v0` has had no commit since March 2018, and its last tag is `v0.9.1`.
Its repository is not archived. Replacements were looked at: shopspring,
cockroachdb/apd and govalues decimal libraries, and a decimal type of the
driver's own, built on `math/big` with `MarshalCQL` and `UnmarshalCQL`. Gemini
proposed the second. Ken chose to keep `inf.v0`, because gocql brings it in
and a replacement adds code or a dependency and removes neither.

The aim is to remove the dependency when gocql allows it. If gocql changes
how it reads or writes a `decimal`, for example to take a type from the
standard library or to stop requiring `inf.Dec`, remove every import of
`inf.v0` from this module. W19 tracks this.
