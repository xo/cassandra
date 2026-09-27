package cql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// pingStmt is the statement that Ping runs. Every Cassandra release has the
// table and the column.
const pingStmt = "SELECT release_version FROM system.local"

// conn is a connection over the session of a Connector. It holds no state of
// its own, because Cassandra keeps no state per connection that database/sql
// can see. gocql refuses a USE statement, so no statement can change the
// keyspace of later statements.
type conn struct {
	c *Connector
	s session

	// owned is true when the connection came from Driver.Open. Such a
	// connection has a Connector of its own, and Close closes it.
	owned bool
}

// Prepare returns a statement for query. It sends nothing to the server,
// because gocql prepares each statement when it first runs and caches it.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return &stmt{c: c, query: query}, nil
}

// PrepareContext returns a statement for query. It sends nothing to the
// server, so an error in query arrives from the first ExecContext or
// QueryContext.
func (c *conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.Prepare(query)
}

// Close closes the connection. The session belongs to the Connector, so Close
// leaves it open, unless the connection came from Driver.Open.
func (c *conn) Close() error {
	if c.owned {
		return c.c.Close()
	}
	return nil
}

// Begin returns ErrNoTransactions.
func (c *conn) Begin() (driver.Tx, error) {
	return nil, ErrNoTransactions
}

// BeginTx returns ErrNoTransactions.
func (c *conn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return nil, ErrNoTransactions
}

// Ping reads the release version of the node that answers.
func (c *conn) Ping(ctx context.Context) error {
	if err := c.s.exec(ctx, pingStmt, nil, options{}); err != nil {
		return fmt.Errorf("pinging cluster: %w", err)
	}
	return nil
}

// ExecContext runs query and discards any rows. It returns
// driver.ResultNoRows, because Cassandra reports no row count and no insert
// ID.
func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	o, values := splitArgs(ctx, args)
	if err := c.s.exec(ctx, query, values, o); err != nil {
		return nil, fmt.Errorf("executing statement: %w", err)
	}
	return driver.ResultNoRows, nil
}

// QueryContext runs query and returns its rows. A statement that the server
// refuses fails here, and not while the rows are read.
func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	o, values := splitArgs(ctx, args)
	it, err := c.s.query(ctx, query, values, o)
	if err != nil {
		return nil, fmt.Errorf("running query: %w", err)
	}
	return newRows(it), nil
}

// CheckNamedValue decides which arguments go to gocql unchanged. database/sql
// calls it for each argument, and it calls driver.Valuer only when
// CheckNamedValue returns driver.ErrSkip.
func (c *conn) CheckNamedValue(nv *driver.NamedValue) error {
	if nv.Name != "" {
		return fmt.Errorf("%w: %s", ErrNamedArgs, nv.Name)
	}
	switch v := nv.Value.(type) {
	case uuid.UUID:
		// gocql takes a gocql.UUID and not the standard type. Both are a
		// [16]byte, so the conversion copies nothing (D25).
		nv.Value = gocql.UUID(v)
		return nil
	case *uuid.UUID:
		nv.Value = (*gocql.UUID)(v)
		return nil
	case sql.Null[uuid.UUID]:
		// Its Value method returns a uuid.UUID, which database/sql refuses.
		nv.Value = nil
		if v.Valid {
			nv.Value = gocql.UUID(v.V)
		}
		return nil
	case nil, Option, gocql.Marshaler:
		// An Option is taken out of the arguments by splitArgs. A Marshaler
		// comes before a Valuer, so that a type with both uses its CQL form.
		return nil
	case driver.Valuer:
		return driver.ErrSkip
	}
	switch reflect.ValueOf(nv.Value).Kind() {
	case reflect.Chan, reflect.Func, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return fmt.Errorf("%w: %T", ErrUnsupportedArg, nv.Value)
	}
	// gocql marshals the value when the statement runs, and it returns an
	// error that names both types when the value does not fit the column.
	return nil
}

// ResetSession returns driver.ErrBadConn after the Connector is closed, so
// that database/sql discards the connection.
func (c *conn) ResetSession(context.Context) error {
	if c.c.isClosed() {
		return driver.ErrBadConn
	}
	return nil
}

// IsValid reports whether the Connector is still open.
func (c *conn) IsValid() bool {
	return !c.c.isClosed()
}
