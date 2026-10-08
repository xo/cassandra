# D23. A query option can come from an argument or from the context

Status: Decided. Amended by D33.

Ken decided this on 2026-09-27. It answers what was open question 7.

Every query option type, such as `cql.PageSize`, satisfies one unexported
interface. A caller can pass an option in the argument list:

```go
rows, err := db.QueryContext(ctx, q, id, cql.PageSize(500))
```

A caller can also attach options to a context, and every call that uses the
context gets them:

```go
ctx = cql.WithOptions(ctx, cql.Consistency(gocql.LocalQuorum))
```

For each call, the driver applies the options from the context first, then
the options from the arguments. So an argument overrides the context. The
`conn` stores nothing between calls.

Gemini proposed arguments and DeepSeek proposed the context. `n1ql` has the
same question as its Q10, and it decides its own answer.
