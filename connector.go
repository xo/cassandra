package cassandra

import (
	"context"
	"database/sql/driver"
	"fmt"
	"slices"
	"sync"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// Connector opens connections to one Cassandra cluster. Every connection
// shares one gocql session, which the Connector creates on the first Connect.
// Close the Connector to close the session. sql.DB.Close closes it.
//
// Use a Connector to set a gocql option that the DSN cannot express, such as
// a host selection policy, a retry policy, a logger or an observer:
//
//	cfg := gocql.NewCluster("10.0.0.1", "10.0.0.2")
//	cfg.Logger = logger
//	db := sql.OpenDB(cassandra.NewConnector(cfg))
type Connector struct {
	cfg        gocql.ClusterConfig
	newSession func(gocql.ClusterConfig) (session, error)

	mu     sync.Mutex
	sess   session
	closed bool
}

// NewConnector returns a Connector for cfg. It keeps a copy of cfg, so a
// change to cfg after the call does not reach the Connector. It does not
// connect.
func NewConnector(cfg *gocql.ClusterConfig) *Connector {
	c := *cfg
	c.Hosts = slices.Clone(cfg.Hosts)
	if cfg.SslOpts != nil {
		opts := *cfg.SslOpts
		c.SslOpts = &opts
	}
	return &Connector{
		cfg:        c,
		newSession: newSession,
	}
}

// Connect returns a connection over the shared session. The first call
// creates the session. When that fails, Connect returns the error, and the
// next call tries again.
//
// gocql creates a session with no context, so the connectTimeout key of the
// DSN limits how long the first call takes.
func (c *Connector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.connect()
}

// Driver returns the cassandra driver.
func (c *Connector) Driver() driver.Driver {
	return Driver{}
}

// Close closes the shared session. A connection that is open stops working.
// Close can be called more than once.
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.sess != nil {
		c.sess.close()
		c.sess = nil
	}
	return nil
}

// connect returns a connection over the shared session, and creates the
// session if there is none.
func (c *Connector) connect() (*conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, ErrConnectorClosed
	}
	if c.sess == nil {
		sess, err := c.newSession(c.cfg)
		if err != nil {
			return nil, fmt.Errorf("creating session: %w", err)
		}
		c.sess = sess
	}
	return &conn{c: c, s: c.sess}, nil
}

// isClosed reports whether Close was called.
func (c *Connector) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
