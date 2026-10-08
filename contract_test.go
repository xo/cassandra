package cassandra //nolint:testpackage // The contract runs over the fake session, which only the package can set.

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp"
	"github.com/xo/dbimp/dbimptest"
)

// waitSession is a session whose statements end only when their context ends.
type waitSession struct {
	fakeSession
}

func (s *waitSession) exec(ctx context.Context, _ string, _ []any, _ options) error {
	<-ctx.Done()
	return ctx.Err()
}

func (s *waitSession) query(ctx context.Context, _ string, _ []any, _ options) (iterator, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestContract holds the contract that every xo driver keeps (dbimp step 11),
// over the fake session. The contract of dbimptest.RunContract serves a
// response over HTTP, so it does not fit a driver of the native protocol (D30).
// It must not run in parallel, because CheckGoroutines counts the goroutines
// of the process.
func TestContract(t *testing.T) {
	dbimptest.CheckGoroutines(t)
	t.Run("the order of the columns", func(t *testing.T) {
		s := &fakeSession{result: result{
			cols: []gocql.ColumnInfo{column("b", native(gocql.TypeInt)), column("a", native(gocql.TypeInt)), column("c", native(gocql.TypeInt))},
			rows: [][]any{{2, 1, 3}},
		}}
		rows, err := newTestDB(t, s).QueryContext(t.Context(), "SELECT b, a, c FROM t")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		cols, err := rows.Columns()
		if err != nil || len(cols) != 3 || cols[0] != "b" || cols[1] != "a" || cols[2] != "c" {
			t.Errorf("got %v and %v, want b, a, c", cols, err)
		}
	})
	t.Run("NULL is nil", func(t *testing.T) {
		s := &fakeSession{result: result{
			cols: []gocql.ColumnInfo{column("a", native(gocql.TypeInt)), column("b", native(gocql.TypeText))},
			rows: [][]any{{1, nil}},
		}}
		var a, b any
		if err := newTestDB(t, s).QueryRowContext(t.Context(), "SELECT a, b FROM t").Scan(&a, &b); err != nil {
			t.Fatal(err)
		}
		if b != nil {
			t.Errorf("got %#v for NULL, want nil", b)
		}
	})
	t.Run("an error after some rows", func(t *testing.T) {
		boom := errors.New("boom")
		s := &fakeSession{result: result{
			cols: []gocql.ColumnInfo{column("a", native(gocql.TypeInt))},
			rows: [][]any{{1}, {2}, {3}},
			err:  boom,
		}}
		rows, err := newTestDB(t, s).QueryContext(t.Context(), "SELECT a FROM t")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		n := 0
		for rows.Next() {
			n++
		}
		if n != 3 || !errors.Is(rows.Err(), boom) || !errors.Is(rows.Err(), dbimp.ErrIncomplete) {
			t.Errorf("got %d rows and %v, want 3 rows and an error that wraps boom and dbimp.ErrIncomplete", n, rows.Err())
		}
	})
	t.Run("a close before the end", func(t *testing.T) {
		s := &fakeSession{result: result{
			cols: []gocql.ColumnInfo{column("a", native(gocql.TypeInt))},
			rows: [][]any{{1}, {2}, {3}},
		}}
		rows, err := newTestDB(t, s).QueryContext(t.Context(), "SELECT a FROM t")
		if err != nil {
			t.Fatal(err)
		}
		if !rows.Next() {
			t.Fatalf("got no row: %v", rows.Err())
		}
		if err := rows.Close(); err != nil {
			t.Errorf("closing before the end: %v", err)
		}
	})
	t.Run("a cancelled context", func(t *testing.T) {
		db := sql.OpenDB(newTestConnector(&waitSession{}))
		defer db.Close()
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()
		_, err := db.ExecContext(ctx, "DELETE FROM t WHERE id = 1")
		if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, driver.ErrBadConn) {
			t.Errorf("got %v, want an error that wraps context.DeadlineExceeded and never driver.ErrBadConn", err)
		}
		if _, err := db.QueryContext(ctx, "SELECT id FROM t"); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("got %v, want an error that wraps context.DeadlineExceeded", err)
		}
	})
	t.Run("a statement is sent once", func(t *testing.T) {
		s := &fakeSession{err: gocql.ErrNoConnections}
		_, err := newTestDB(t, s).ExecContext(t.Context(), "INSERT INTO t (id) VALUES (1)")
		if !errors.Is(err, gocql.ErrNoConnections) || errors.Is(err, driver.ErrBadConn) {
			t.Errorf("got %v, want the error of gocql and never driver.ErrBadConn", err)
		}
		if n := len(s.calls); n != 1 {
			t.Errorf("the session received %d statements, want 1", n)
		}
	})
	t.Run("no transaction", func(t *testing.T) {
		_, err := newTestDB(t, &fakeSession{}).BeginTx(t.Context(), nil)
		if !errors.Is(err, dbimp.ErrNotSupported) {
			t.Errorf("got %v, want an error that wraps dbimp.ErrNotSupported", err)
		}
	})
}
