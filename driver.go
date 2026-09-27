// Package cql is a database/sql driver for Apache Cassandra. It wraps
// github.com/apache/cassandra-gocql-driver/v2 and registers itself as "cql".
//
// # Connecting
//
// Open a database with a DSN in one of two forms:
//
//	db, err := sql.Open("cql", "cql://user:password@10.0.0.1,10.0.0.2:9042/keyspace?consistency=localQuorum")
//	db, err := sql.Open("cql", "10.0.0.1,10.0.0.2:9042?keyspace=keyspace&username=user&password=password")
//
// A DSN that starts with cql:// or cassandra:// is a URL. Any other DSN is a
// list of hosts, separated by commas, and a query string. See ParseDSN for
// every key. To set a gocql option that the DSN cannot express, build a
// gocql.ClusterConfig and pass it to NewConnector.
//
// All the connections of one sql.DB share one gocql session, which holds its
// own pool of connections to each node. The pool settings of sql.DB, such as
// SetMaxOpenConns, limit how many statements run at the same time. They do not
// limit the connections to Cassandra. The numConns key does that.
//
// # Arguments
//
// A statement takes its values at ? markers. Every Go value that gocql can
// marshal is accepted, including slices, maps, gocql.UUID, gocql.Duration,
// *big.Int, *inf.Dec, net.IP, structs for user defined types, and
// gocql.UnsetValue. Named arguments from sql.Named are refused, because gocql
// cannot bind a value by name.
//
// # Query options
//
// An Option, such as Consistency or PageSize, changes how one statement runs.
// Pass it among the arguments, or attach it to a context with WithOptions:
//
//	rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE org = ?", org, cql.PageSize(500))
//
// # Scanning
//
// Scan a column into any type that gocql can decode it into, such as
// *[]string, *map[string]int or *gocql.UUID, or into any type that
// database/sql can convert to. A uuid column is the standard uuid.UUID, and
// the driver takes a uuid.UUID argument too. A NULL scanned into a plain *string or *int64
// is an error, as it is with other drivers. Scan into a sql.Null[T], a
// pointer to a pointer, or *any to read a column that can be NULL. Cassandra
// stores an empty collection as NULL, so a NULL scanned into a slice or a map
// gives nil and no error.
//
// # Transactions, batches and lightweight transactions
//
// CQL has no transactions, and BeginTx returns ErrNoTransactions. A batch is a
// CQL statement. Pass the whole BEGIN BATCH ... APPLY BATCH text to
// ExecContext. To read the [applied] column of a lightweight transaction, run
// it with QueryContext and scan the first column into a bool. When it is
// false, the row also holds the values that are there already, one column
// each.
//
// ExecContext returns driver.ResultNoRows, because Cassandra reports no row
// count and no insert ID.
package cql

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
)

func init() {
	sql.Register("cql", Driver{})
}

// Driver is the cql driver. It holds no configuration.
type Driver struct{}

// Open returns a connection with its own session. database/sql does not call
// Open, because Driver implements driver.DriverContext. Closing the connection
// closes its session.
func (d Driver) Open(dsn string) (driver.Conn, error) {
	c, err := d.openConnector(dsn)
	if err != nil {
		return nil, err
	}
	cn, err := c.connect()
	if err != nil {
		return nil, err
	}
	cn.owned = true
	return cn, nil
}

// OpenConnector parses dsn and returns a Connector for it. It does not
// connect.
func (d Driver) OpenConnector(dsn string) (driver.Connector, error) {
	return d.openConnector(dsn)
}

func (Driver) openConnector(dsn string) (*Connector, error) {
	cfg, err := ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("opening connector: %w", err)
	}
	return NewConnector(cfg), nil
}
