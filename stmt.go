package cql

import (
	"context"
	"database/sql/driver"
)

// stmt is a statement that a connection prepared. It holds only the text,
// because gocql prepares and caches the statement itself when it runs.
type stmt struct {
	c     *conn
	query string
}

// Close returns nil. The statement holds nothing to release.
func (s *stmt) Close() error {
	return nil
}

// NumInput returns -1. The driver cannot count the ? markers without parsing
// CQL, because a ? can appear inside a string literal.
func (s *stmt) NumInput() int {
	return -1
}

// Exec returns ErrNoContext. database/sql calls ExecContext instead.
func (s *stmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, ErrNoContext
}

// Query returns ErrNoContext. database/sql calls QueryContext instead.
func (s *stmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, ErrNoContext
}

// ExecContext runs the statement. See conn.ExecContext.
func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.c.ExecContext(ctx, s.query, args)
}

// QueryContext runs the statement and returns its rows. See conn.QueryContext.
func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.c.QueryContext(ctx, s.query, args)
}
