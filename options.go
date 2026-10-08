package cassandra

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp"
)

// Option changes how one statement runs (dbimp D109). Pass an Option as an
// argument to a query, or attach it to a context with WithOptions. An Option
// in the argument list is not sent to Cassandra as a value.
//
// The DSN sets the defaults for the whole session. An Option from the context
// overrides the DSN, and an Option from the arguments overrides the context.
// When two options set the same thing, the later one wins.
type Option = dbimp.Option[options]

// options holds the options of one statement. A zero field leaves the default
// of the session in place.
type options struct {
	timeout           time.Duration
	readonly          bool
	params            map[string]any
	database          string
	consistency       *gocql.Consistency
	serialConsistency *gocql.Consistency
	pageSize          int
	idempotent        *bool
	timestamp         *time.Time
}

// WithOptions returns a copy of ctx that carries opts. Every statement that
// runs with the returned context uses them. When ctx already carries options,
// opts come after them, so a later option wins.
func WithOptions(ctx context.Context, opts ...Option) context.Context {
	return dbimp.WithOptions(ctx, opts...)
}

// WithTimeout ends the statement after d, which includes the time that it
// takes to read every page of its rows. Zero sets no timeout of its own, and
// the timeout key of the DSN then applies to each request.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithReadonly asks the server to refuse a write. Cassandra has no read-only
// statement, so WithReadonly(true) fails the statement with an error that
// wraps dbimp.ErrNotSupported. Create a role that can only read, and connect
// as that role.
func WithReadonly(readonly bool) Option {
	return func(o *options) { o.readonly = readonly }
}

// WithParameter sets a key of the request by its name. The native protocol has
// no body of keys for a statement, so any name fails the statement with an
// error that wraps dbimp.ErrNotSupported. Use the option of the driver for
// each setting that gocql lets one statement change.
func WithParameter(name string, value any) Option {
	return func(o *options) {
		if o.params == nil {
			o.params = map[string]any{}
		}
		o.params[name] = value
	}
}

// WithDatabase sets the keyspace of the statement. gocql sends it with the
// statement, which needs protocol version 5, and a server that speaks an older
// version refuses the statement.
func WithDatabase(name string) Option {
	return func(o *options) { o.database = name }
}

// WithConsistency sets the consistency level of the statement, as the key
// consistency of the DSN does for the session.
func WithConsistency(c gocql.Consistency) Option {
	return func(o *options) { o.consistency = &c }
}

// WithSerialConsistency sets the serial consistency level of a lightweight
// transaction.
func WithSerialConsistency(c gocql.Consistency) Option {
	return func(o *options) { o.serialConsistency = &c }
}

// WithPageSize sets how many rows gocql reads in one page. gocql reads the next
// page when the rows of the current page are used up.
func WithPageSize(n int) Option {
	return func(o *options) { o.pageSize = n }
}

// WithIdempotent tells gocql whether the statement is safe to run more than
// once. gocql retries a statement only when it is idempotent.
func WithIdempotent(idempotent bool) Option {
	return func(o *options) { o.idempotent = &idempotent }
}

// WithTimestamp sets the write time of the statement. Cassandra stores the
// time in microseconds.
func WithTimestamp(t time.Time) Option {
	return func(o *options) { o.timestamp = &t }
}

// resolve returns the options of one statement, those of the context and then
// the Option arguments of args, and the values to bind, in their order. It
// returns an error for an option that the server cannot honor, or whose value
// the DSN refuses (dbimp D109).
func resolve(ctx context.Context, args []driver.NamedValue) (options, []any, error) {
	o, rest := dbimp.Resolve(ctx, options{}, args)
	if err := o.check(); err != nil {
		return options{}, nil, err
	}
	values := make([]any, len(rest))
	for i, arg := range rest {
		values[i] = arg.Value
	}
	return o, values, nil
}

// check returns an error for an option that the server cannot honor, or whose
// value the DSN refuses.
func (o options) check() error {
	switch {
	case o.timeout < 0:
		return fmt.Errorf("applying the option WithTimeout: %v is negative: %w", o.timeout, dbimp.ErrInvalidValue)
	case o.pageSize < 0:
		return fmt.Errorf("applying the option WithPageSize: %d is negative: %w", o.pageSize, dbimp.ErrInvalidValue)
	case o.readonly:
		return dbimp.Unsupported("WithReadonly")
	case len(o.params) > 0:
		return dbimp.Unsupported("WithParameter")
	}
	return nil
}

// set applies o to q.
func (o options) set(q *gocql.Query) *gocql.Query {
	if o.consistency != nil {
		q = q.Consistency(*o.consistency)
	}
	if o.serialConsistency != nil {
		q = q.SerialConsistency(*o.serialConsistency)
	}
	if o.pageSize > 0 {
		q = q.PageSize(o.pageSize)
	}
	if o.idempotent != nil {
		q = q.Idempotent(*o.idempotent)
	}
	if o.timestamp != nil {
		q = q.WithTimestamp(o.timestamp.UnixMicro())
	}
	if o.database != "" {
		q = q.SetKeyspace(o.database)
	}
	return q
}

// withTimeout returns ctx with the timeout of o, and a function that releases
// it. With no timeout, it returns ctx and a function that does nothing.
func (o options) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if o.timeout > 0 {
		return context.WithTimeout(ctx, o.timeout)
	}
	return ctx, func() {}
}
