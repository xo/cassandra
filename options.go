package cql

import (
	"context"
	"database/sql/driver"
	"slices"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// Option changes how one statement runs. Pass an Option as an argument to a
// query, or attach it to a context with WithOptions. An Option in the argument
// list is not sent to Cassandra as a value.
//
// The DSN sets the defaults for the whole session. An Option from the context
// overrides the DSN, and an Option from the arguments overrides the context.
type Option interface {
	apply(o *options)
}

// Consistency sets the consistency level of one statement.
type Consistency gocql.Consistency

// SerialConsistency sets the serial consistency level of one lightweight
// transaction.
type SerialConsistency gocql.Consistency

// PageSize sets how many rows gocql reads in one page. gocql reads the next
// page when the rows of the current page are used up.
type PageSize int

// Idempotent tells gocql whether the statement is safe to run more than once.
// gocql retries a statement only when it is idempotent.
type Idempotent bool

// Timestamp sets the write time of one statement. Cassandra stores the time
// in microseconds.
type Timestamp time.Time

func (c Consistency) apply(o *options) {
	v := gocql.Consistency(c)
	o.consistency = &v
}

func (c SerialConsistency) apply(o *options) {
	v := gocql.Consistency(c)
	o.serialConsistency = &v
}

func (n PageSize) apply(o *options) {
	v := int(n)
	o.pageSize = &v
}

func (b Idempotent) apply(o *options) {
	v := bool(b)
	o.idempotent = &v
}

func (t Timestamp) apply(o *options) {
	v := time.Time(t)
	o.timestamp = &v
}

// options holds the query options for one statement. A nil field leaves the
// session default in place.
type options struct {
	consistency       *gocql.Consistency
	serialConsistency *gocql.Consistency
	pageSize          *int
	idempotent        *bool
	timestamp         *time.Time
}

// set applies o to q.
func (o options) set(q *gocql.Query) *gocql.Query {
	if o.consistency != nil {
		q = q.Consistency(*o.consistency)
	}
	if o.serialConsistency != nil {
		q = q.SerialConsistency(*o.serialConsistency)
	}
	if o.pageSize != nil {
		q = q.PageSize(*o.pageSize)
	}
	if o.idempotent != nil {
		q = q.Idempotent(*o.idempotent)
	}
	if o.timestamp != nil {
		q = q.WithTimestamp(o.timestamp.UnixMicro())
	}
	return q
}

// optionsKey is the context key for the options that WithOptions attaches.
type optionsKey struct{}

// WithOptions returns a copy of ctx that carries opts. Every statement that
// runs with the returned context uses them. When ctx already carries options,
// opts come after them, so a later option of the same type wins.
func WithOptions(ctx context.Context, opts ...Option) context.Context {
	prev, _ := ctx.Value(optionsKey{}).([]Option)
	return context.WithValue(ctx, optionsKey{}, append(slices.Clip(prev), opts...))
}

// splitArgs separates the query options in args from the values to bind. The
// options from ctx apply first, then the options from args. The values keep
// their order.
func splitArgs(ctx context.Context, args []driver.NamedValue) (options, []any) {
	var o options
	if opts, ok := ctx.Value(optionsKey{}).([]Option); ok {
		for _, opt := range opts {
			opt.apply(&o)
		}
	}
	values := make([]any, 0, len(args))
	for _, arg := range args {
		if opt, ok := arg.Value.(Option); ok {
			opt.apply(&o)
			continue
		}
		values = append(values, arg.Value)
	}
	return o, values
}
