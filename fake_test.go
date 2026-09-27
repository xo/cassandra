package cql

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// call is one statement that a fake session received.
type call struct {
	stmt   string
	values []any
	opts   options
}

// result is what a fake session returns for a query.
type result struct {
	cols []gocql.ColumnInfo
	// rows holds the Go values of each row. A nil value is NULL, and empty{}
	// is an empty value that is not NULL. The fake marshals each value with
	// gocql, so the decode path of a test is the real gocql code.
	rows [][]any
	// err is what close returns after the last row.
	err error
}

// empty is a value of zero bytes that is not NULL.
type empty struct{}

// fakeSession is a session that records each statement and returns a result
// that the test sets up.
type fakeSession struct {
	mu     sync.Mutex
	calls  []call
	err    error // returned by exec and query
	result result
	closed int
}

func (s *fakeSession) record(stmt string, values []any, o options) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, call{stmt: stmt, values: values, opts: o})
}

func (s *fakeSession) exec(_ context.Context, stmt string, values []any, o options) error {
	s.record(stmt, values, o)
	return s.err
}

func (s *fakeSession) query(_ context.Context, stmt string, values []any, o options) (iterator, error) {
	s.record(stmt, values, o)
	if s.err != nil {
		return nil, s.err
	}
	return &fakeIter{res: s.result}, nil
}

func (s *fakeSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed++
}

// lastCall returns the last statement that s received.
func (s *fakeSession) lastCall(t *testing.T) call {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls) == 0 {
		t.Fatal("the session received no statement")
	}
	return s.calls[len(s.calls)-1]
}

// fakeIter returns the rows of a result the way gocql does: it marshals each
// value and hands the bytes to the scan destinations, and it expands a tuple
// column into one destination for each element.
type fakeIter struct {
	res result
	pos int
	err error
}

func (it *fakeIter) columns() []gocql.ColumnInfo {
	return it.res.cols
}

func (it *fakeIter) scan(dest ...any) bool {
	if it.err != nil || it.pos >= len(it.res.rows) {
		return false
	}
	row := it.res.rows[it.pos]
	it.pos++
	i := 0
	for j, col := range it.res.cols {
		data, err := marshal(col.TypeInfo, row[j])
		if err != nil {
			it.err = err
			return false
		}
		if t, ok := col.TypeInfo.(gocql.TupleTypeInfo); ok {
			n := len(t.Elems)
			err = gocql.Unmarshal(col.TypeInfo, data, dest[i:i+n])
			i += n
		} else {
			err = gocql.Unmarshal(col.TypeInfo, data, dest[i])
			i++
		}
		if err != nil {
			it.err = err
			return false
		}
	}
	return true
}

func (it *fakeIter) close() error {
	if it.err != nil {
		return it.err
	}
	return it.res.err
}

// marshal returns the wire bytes of v for info.
func marshal(info gocql.TypeInfo, v any) ([]byte, error) {
	switch v.(type) {
	case nil:
		return nil, nil
	case empty:
		return []byte{}, nil
	}
	return gocql.Marshal(info, v)
}

// native returns the type of a native CQL column.
func native(t gocql.Type) gocql.TypeInfo {
	return gocql.NewNativeType(4, t, "")
}

// parse returns the type that gocql reads from a CQL type name, such as
// map<text, int>.
func parse(name string) gocql.TypeInfo {
	return gocql.NewNativeType(4, gocql.TypeCustom, name)
}

// column returns a column of the result.
func column(name string, info gocql.TypeInfo) gocql.ColumnInfo {
	return gocql.ColumnInfo{Keyspace: "ks", Table: "t", Name: name, TypeInfo: info}
}

// newTestConnector returns a Connector whose session is s.
func newTestConnector(s session) *Connector {
	c := NewConnector(gocql.NewCluster("127.0.0.1"))
	c.newSession = func(gocql.ClusterConfig) (session, error) {
		return s, nil
	}
	return c
}

// newTestDB returns a database over s. The test closes it.
func newTestDB(t *testing.T, s session) *sql.DB {
	t.Helper()
	db := sql.OpenDB(newTestConnector(s))
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing db: %v", err)
		}
	})
	return db
}
