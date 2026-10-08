package cassandra

import (
	"errors"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp"
)

// TestOptionsPrecedence holds dbimp D109: an option from the context comes
// first, an option from an argument overrides it, and a later option wins.
func TestOptionsPrecedence(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	db := newTestDB(t, s)
	ctx := WithOptions(t.Context(), WithConsistency(gocql.One), WithPageSize(5))
	ctx = WithOptions(ctx, WithPageSize(10))
	if _, err := db.ExecContext(ctx, "DELETE FROM t WHERE id = ?", 1, WithConsistency(gocql.All)); err != nil {
		t.Fatal(err)
	}
	got := s.lastCall(t).opts
	if *got.consistency != gocql.All || got.pageSize != 10 {
		t.Errorf("got consistency %v and page size %d, want ALL and 10", *got.consistency, got.pageSize)
	}
}

// TestWithTimeout holds that the timeout of an option ends the statement, for
// Exec and for Query, and that no option leaves the deadline to the caller.
func TestWithTimeout(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	db := newTestDB(t, s)
	if _, err := db.ExecContext(t.Context(), "DELETE FROM t WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	if d := s.lastCall(t).deadline; !d.IsZero() {
		t.Errorf("got the deadline %v with no timeout, want none", d)
	}
	start := time.Now()
	if _, err := db.ExecContext(t.Context(), "DELETE FROM t WHERE id = 1", WithTimeout(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if d := s.lastCall(t).deadline; d.Before(start.Add(time.Minute - time.Second)) {
		t.Errorf("got the deadline %v, want about a minute from %v", d, start)
	}
	rows, err := db.QueryContext(t.Context(), "SELECT id FROM t", WithTimeout(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if d := s.lastCall(t).deadline; d.IsZero() {
		t.Error("got no deadline for a query with a timeout")
	}
}

// TestOptionsTheServerCannotHonor holds dbimp D109: an option that Cassandra
// cannot honor fails the statement with dbimp.ErrNotSupported, and sends
// nothing. An option that asks for nothing does not fail.
func TestOptionsTheServerCannotHonor(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		opt  Option
		want error
	}{
		{"readonly", WithReadonly(true), dbimp.ErrNotSupported},
		{"not readonly", WithReadonly(false), nil},
		{"parameter", WithParameter("x", 1), dbimp.ErrNotSupported},
		{"negative timeout", WithTimeout(-time.Second), dbimp.ErrInvalidValue},
		{"zero timeout", WithTimeout(0), nil},
		{"negative page size", WithPageSize(-1), dbimp.ErrInvalidValue},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &fakeSession{}
			db := newTestDB(t, s)
			_, err := db.ExecContext(t.Context(), "DELETE FROM t WHERE id = 1", tt.opt)
			if !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if sent := len(s.calls); (tt.want == nil) != (sent == 1) {
				t.Errorf("the session received %d statements, want one only when the option is honored", sent)
			}
		})
	}
}

// TestWithDatabase holds that the keyspace of an option reaches the session.
func TestWithDatabase(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	db := newTestDB(t, s)
	if _, err := db.ExecContext(t.Context(), "DELETE FROM t WHERE id = 1", WithDatabase("other")); err != nil {
		t.Fatal(err)
	}
	if got := s.lastCall(t).opts.database; got != "other" {
		t.Errorf("got the keyspace %q, want other", got)
	}
}
