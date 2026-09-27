package cql

import (
	"context"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// session is the part of a gocql session that the driver uses. The unit tests
// replace it with a fake.
type session interface {
	// exec runs stmt and discards any rows.
	exec(ctx context.Context, stmt string, values []any, o options) error
	// query runs stmt and returns its rows. It returns the error from the
	// server when the server refuses stmt.
	query(ctx context.Context, stmt string, values []any, o options) (iterator, error)
	// close closes the session.
	close()
}

// iterator is the part of a gocql iterator that the driver uses.
type iterator interface {
	// columns returns the columns of the result.
	columns() []gocql.ColumnInfo
	// scan reads the next row into dest. It returns false at the end of the
	// rows or on an error, and close then returns the error.
	scan(dest ...any) bool
	// close releases the iterator and returns the first error it met.
	close() error
}

// newSession creates a gocql session for cfg.
func newSession(cfg gocql.ClusterConfig) (session, error) {
	s, err := cfg.CreateSession()
	if err != nil {
		return nil, err
	}
	return gocqlSession{s: s}, nil
}

// gocqlSession is a session over a gocql session.
type gocqlSession struct {
	s *gocql.Session
}

func (s gocqlSession) exec(ctx context.Context, stmt string, values []any, o options) error {
	return o.set(s.s.Query(stmt, values...)).ExecContext(ctx)
}

func (s gocqlSession) query(ctx context.Context, stmt string, values []any, o options) (iterator, error) {
	it := o.set(s.s.Query(stmt, values...)).IterContext(ctx)
	// gocql sends the request and reads the first page when it creates the
	// iterator. RowData returns the error of the iterator and reads no row.
	if _, err := it.RowData(); err != nil {
		_ = it.Close()
		return nil, err
	}
	return gocqlIter{it: it}, nil
}

func (s gocqlSession) close() {
	s.s.Close()
}

// gocqlIter is an iterator over a gocql iterator.
type gocqlIter struct {
	it *gocql.Iter
}

func (it gocqlIter) columns() []gocql.ColumnInfo {
	return it.it.Columns()
}

func (it gocqlIter) scan(dest ...any) bool {
	return it.it.Scan(dest...)
}

func (it gocqlIter) close() error {
	return it.it.Close()
}
