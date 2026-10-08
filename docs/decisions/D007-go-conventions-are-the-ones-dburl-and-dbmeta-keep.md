# D7. Go conventions are the ones dburl and dbmeta keep

Status: Decided.

- Sentinel errors are constants of `type Error string`, never variables built
  with `errors.New` or `fmt.Errorf`. An importer can reassign a variable but
  not a constant.
- Every error is wrapped with `%w`. A message is lower case, starts with a
  gerund, and names the object that failed. It does not say "failed to" or
  "error".
- `context.Context` comes first, is named `ctx`, and is never stored in a
  struct.
- `any` instead of `interface{}`, `io` instead of `io/ioutil`.
- A receiver is named with one or two lower case letters, the same letter on
  every method of a type.

`config.go`, `connection.go` and `globals.go` break each of these today.
