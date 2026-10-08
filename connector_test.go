package cassandra

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp"
)

func TestConnectorSharesOneSession(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	c := newTestConnector(s)
	var mu sync.Mutex
	created := 0
	c.newSession = func(gocql.ClusterConfig) (session, error) {
		mu.Lock()
		defer mu.Unlock()
		created++
		return s, nil
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			cn, err := c.Connect(t.Context())
			if err != nil {
				t.Error(err)
				return
			}
			if err := cn.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if created != 1 {
		t.Errorf("created %d sessions, want 1", created)
	}
	if s.closed != 0 {
		t.Errorf("closing a connection closed the session")
	}
}

func TestConnectorTriesAgainAfterAFailure(t *testing.T) {
	t.Parallel()
	fault := errors.New("no hosts")
	s := &fakeSession{}
	c := newTestConnector(s)
	c.newSession = func(gocql.ClusterConfig) (session, error) {
		return nil, fault
	}
	if _, err := c.Connect(t.Context()); !errors.Is(err, fault) {
		t.Fatalf("got %v, want %v", err, fault)
	}
	c.newSession = func(gocql.ClusterConfig) (session, error) {
		return s, nil
	}
	if _, err := c.Connect(t.Context()); err != nil {
		t.Fatalf("second try: %v", err)
	}
}

func TestConnectorClose(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	c := newTestConnector(s)
	dc, err := c.Connect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	cn, ok := dc.(*conn)
	if !ok {
		t.Fatalf("got %T, want *conn", dc)
	}
	if !cn.IsValid() || cn.ResetSession(t.Context()) != nil {
		t.Fatal("an open connection is not valid")
	}
	for range 2 {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if s.closed != 1 {
		t.Errorf("closed the session %d times, want 1", s.closed)
	}
	if cn.IsValid() {
		t.Error("IsValid is true after Close")
	}
	if err := cn.ResetSession(t.Context()); !errors.Is(err, driver.ErrBadConn) {
		t.Errorf("ResetSession: got %v, want %v", err, driver.ErrBadConn)
	}
	if _, err := c.Connect(t.Context()); !errors.Is(err, ErrConnectorClosed) {
		t.Errorf("Connect: got %v, want %v", err, ErrConnectorClosed)
	}
}

func TestConnectWithDoneContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := newTestConnector(&fakeSession{}).Connect(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}
}

func TestNewConnectorCopiesTheConfig(t *testing.T) {
	t.Parallel()
	cfg := gocql.NewCluster("a", "b")
	cfg.SslOpts = &gocql.SslOptions{CaPath: "ca"}
	c := NewConnector(cfg)
	cfg.Hosts[0] = "changed"
	cfg.SslOpts.CaPath = "changed"
	cfg.Keyspace = "changed"
	if c.cfg.Hosts[0] != "a" || c.cfg.SslOpts.CaPath != "ca" || c.cfg.Keyspace != "" {
		t.Errorf("a change to the config reached the connector: %v %q %q", c.cfg.Hosts, c.cfg.SslOpts.CaPath, c.cfg.Keyspace)
	}
}

func TestOwnedConnectionClosesItsSession(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	cn, err := newTestConnector(s).connect()
	if err != nil {
		t.Fatal(err)
	}
	cn.owned = true
	if err := cn.Close(); err != nil {
		t.Fatal(err)
	}
	if s.closed != 1 {
		t.Errorf("closed the session %d times, want 1", s.closed)
	}
}

func TestDriverRefusesABadDSN(t *testing.T) {
	t.Parallel()
	if _, err := (Driver{}).Open("cassandra://?nope=1"); !errors.Is(err, dbimp.ErrUnknownKey) {
		t.Errorf("Open: got %v, want %v", err, dbimp.ErrUnknownKey)
	}
	if _, err := (Driver{}).OpenConnector("cassandra://?nope=1"); !errors.Is(err, dbimp.ErrUnknownKey) {
		t.Errorf("OpenConnector: got %v, want %v", err, dbimp.ErrUnknownKey)
	}
	c, err := (Driver{}).OpenConnector("cassandra://h1/ks")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Driver().(Driver); !ok {
		t.Errorf("got driver %T, want Driver", c.Driver())
	}
}

// TestTheDriverRegistersOneName holds hard rule 1: the driver registers the
// name cassandra, and no alias.
func TestTheDriverRegistersOneName(t *testing.T) {
	t.Parallel()
	var found []string
	for _, name := range sql.Drivers() {
		if strings.Contains(name, "cassandra") || strings.Contains(name, "cql") {
			found = append(found, name)
		}
	}
	if len(found) != 1 || found[0] != Name {
		t.Errorf("got the drivers %v, want only %q", found, Name)
	}
}
