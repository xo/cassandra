// Package cassandra is a database/sql driver for Apache Cassandra. It wraps
// github.com/apache/cassandra-gocql-driver/v2 and registers itself as "cassandra".
//
// # Connecting
//
// Open a database with a DSN, which is a URL whose scheme is cassandra:
//
//	db, err := sql.Open("cassandra", "cassandra://user:password@10.0.0.1,10.0.0.2:9042/keyspace?consistency=localQuorum")
//
// The driver reads no other scheme and no older form of the DSN. See ParseDSN
// for every key. To set a gocql option that the DSN cannot express, build a
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
// *big.Int, net.IP, structs for user defined types, and gocql.UnsetValue. The
// canonical types of a column bind too: a dbimp.Date, a dbimp.LocalTime, a
// dbimp.Interval, an *apd.Decimal, a netip.Addr and a dbimp.Vector. Named arguments from sql.Named are refused, because gocql
// cannot bind a value by name.
//
// # Query options
//
// An Option, such as WithConsistency or WithPageSize, changes how one statement runs.
// Pass it among the arguments, or attach it to a context with WithOptions:
//
//	rows, err := db.QueryContext(ctx, "SELECT id FROM users WHERE org = ?", org, cassandra.WithPageSize(500))
//
// # Scanning
//
// Scan a column into any type that gocql can decode it into, such as
// *[]string, *map[string]int or *gocql.UUID, or into any type that
// database/sql can convert to. A column has the Go type that fits its CQL type,
// by the kinds of dbimp: a decimal is an *apd.Decimal, a varint is a *big.Int, a
// date is a dbimp.Date, a time is a dbimp.LocalTime, a duration is a
// dbimp.Interval, an inet is a netip.Addr, and a uuid is the standard uuid.UUID,
// which the driver takes as an argument too. A NULL scanned into a plain *string or *int64
// is an error, as it is with other drivers. Scan into a sql.Null[T], a
// pointer to a pointer, or *any to read a column that can be NULL. Cassandra
// stores an empty collection as NULL, so a NULL scanned into a slice or a map
// gives nil and no error.
//
// # Transactions, batches and lightweight transactions
//
// CQL has no transactions, and BeginTx returns an error that wraps dbimp.ErrNotSupported. A batch is a
// CQL statement. Pass the whole BEGIN BATCH ... APPLY BATCH text to
// ExecContext. To read the [applied] column of a lightweight transaction, run
// it with QueryContext and scan the first column into a bool. When it is
// false, the row also holds the values that are there already, one column
// each.
//
// ExecContext returns driver.ResultNoRows, because Cassandra reports no row
// count and no insert ID.
package cassandra

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
)

// Name is the name that the driver registers with database/sql, and the scheme
// of its DSN.
const Name = "cassandra"

func init() {
	sql.Register(Name, Driver{})
}

// Driver is the cassandra driver. It holds no configuration.
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
