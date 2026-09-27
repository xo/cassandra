package cql

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"testing"
	"time"
	"unsafe"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// marshaler is a type with its own CQL form and a driver.Valuer form. The
// driver must use the CQL form.
type marshaler struct{}

func (marshaler) MarshalCQL(gocql.TypeInfo) ([]byte, error) { return []byte("cql"), nil }
func (marshaler) Value() (driver.Value, error)              { return "valuer", nil }

func TestCheckNamedValue(t *testing.T) {
	t.Parallel()
	var c conn
	tests := []struct {
		name  string
		value any
		named string
		want  error
	}{
		{"named", 1, "id", ErrNamedArgs},
		{"nil", nil, "", nil},
		{"option", PageSize(10), "", nil},
		{"marshaler before valuer", marshaler{}, "", nil},
		{"valuer", sql.Null[int64]{}, "", driver.ErrSkip},
		{"int", 1, "", nil},
		{"slice", []string{"a"}, "", nil},
		{"map", map[string]int{"a": 1}, "", nil},
		{"uuid", gocql.TimeUUID(), "", nil},
		{"duration", gocql.Duration{Days: 1}, "", nil},
		{"unset", gocql.UnsetValue, "", nil},
		{"typed nil pointer", (*string)(nil), "", nil},
		{"struct", struct{ A int }{1}, "", nil},
		{"channel", make(chan int), "", ErrUnsupportedArg},
		{"function", func() {}, "", ErrUnsupportedArg},
		{"complex", complex(1, 2), "", ErrUnsupportedArg},
		{"unsafe pointer", unsafe.Pointer(nil), "", ErrUnsupportedArg},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			nv := &driver.NamedValue{Name: test.named, Ordinal: 1, Value: test.value}
			if err := c.CheckNamedValue(nv); !errors.Is(err, test.want) || (test.want == nil && err != nil) {
				t.Errorf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestExecValues(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	db := newTestDB(t, s)
	id := gocql.TimeUUID()
	std := uuid.NewV7()
	res, err := db.ExecContext(t.Context(), "INSERT INTO t (id, tags, n, v, s, p, a, b) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, []string{"a"}, sql.Null[int64]{V: 5, Valid: true}, sql.Null[int64]{},
		std, (*uuid.UUID)(nil), sql.Null[uuid.UUID]{V: std, Valid: true}, sql.Null[uuid.UUID]{})
	if err != nil {
		t.Fatal(err)
	}
	got := s.lastCall(t)
	// The standard uuid.UUID reaches gocql as a gocql.UUID (D25).
	want := []any{id, []string{"a"}, int64(5), nil, gocql.UUID(std), (*gocql.UUID)(nil), gocql.UUID(std), nil}
	if !reflect.DeepEqual(got.values, want) {
		t.Errorf("got values %#v, want %#v", got.values, want)
	}
	if _, err := res.RowsAffected(); err == nil {
		t.Error("RowsAffected returned no error")
	}
	if _, err := res.LastInsertId(); err == nil {
		t.Error("LastInsertId returned no error")
	}
}

func TestOptions(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	db := newTestDB(t, s)
	ts := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	ctx := WithOptions(t.Context(), Consistency(gocql.One), PageSize(10))
	ctx = WithOptions(ctx, PageSize(20))
	if _, err := db.ExecContext(ctx, "UPDATE t SET v = ? WHERE id = ?", 1,
		Consistency(gocql.All), 2, SerialConsistency(gocql.LocalSerial), Idempotent(true), Timestamp(ts)); err != nil {
		t.Fatal(err)
	}
	got := s.lastCall(t)
	// CheckNamedValue passes an int to gocql unchanged.
	if want := []any{1, 2}; !reflect.DeepEqual(got.values, want) {
		t.Errorf("got values %#v, want %#v", got.values, want)
	}
	// The argument overrides the context, and a later context wins over an
	// earlier one.
	want := options{
		consistency:       new(gocql.All),
		serialConsistency: new(gocql.LocalSerial),
		pageSize:          new(20),
		idempotent:        new(true),
		timestamp:         &ts,
	}
	if !reflect.DeepEqual(got.opts, want) {
		t.Errorf("got options %+v, want %+v", got.opts, want)
	}
}

func TestWithOptionsDoesNotShare(t *testing.T) {
	t.Parallel()
	base := WithOptions(t.Context(), PageSize(1), PageSize(2))
	a := WithOptions(base, Consistency(gocql.One))
	b := WithOptions(base, Consistency(gocql.All))
	oa, _ := splitArgs(a, nil)
	ob, _ := splitArgs(b, nil)
	if *oa.consistency != gocql.One || *ob.consistency != gocql.All {
		t.Errorf("got %v and %v, want ONE and ALL", *oa.consistency, *ob.consistency)
	}
}

func TestRefusedArgs(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	db := newTestDB(t, s)
	if _, err := db.ExecContext(t.Context(), "INSERT INTO t (id) VALUES (:id)", sql.Named("id", 1)); !errors.Is(err, ErrNamedArgs) {
		t.Errorf("got %v, want %v", err, ErrNamedArgs)
	}
	if _, err := db.ExecContext(t.Context(), "INSERT INTO t (id) VALUES (?)", make(chan int)); !errors.Is(err, ErrUnsupportedArg) {
		t.Errorf("got %v, want %v", err, ErrUnsupportedArg)
	}
	if len(s.calls) != 0 {
		t.Errorf("the session received %d statements, want 0", len(s.calls))
	}
}

func TestErrorsReachTheCaller(t *testing.T) {
	t.Parallel()
	fault := &gocql.RequestErrSyntax{}
	s := &fakeSession{err: fault}
	db := newTestDB(t, s)
	if _, err := db.ExecContext(t.Context(), "INSERT INTO"); !errors.Is(err, fault) {
		t.Errorf("exec: got %v, want %v", err, fault)
	}
	if _, err := db.QueryContext(t.Context(), "SELECT *FROM t"); !errors.Is(err, fault) {
		t.Errorf("query: got %v, want %v", err, fault)
	}
	if err := db.PingContext(t.Context()); !errors.Is(err, fault) {
		t.Errorf("ping: got %v, want %v", err, fault)
	}
	var syntax *gocql.RequestErrSyntax
	if _, err := db.ExecContext(t.Context(), "INSERT INTO"); !errors.As(err, &syntax) {
		t.Errorf("errors.As found no *gocql.RequestErrSyntax in %v", err)
	}
	// database/sql retries a statement only on driver.ErrBadConn. The driver
	// never returns it, so each call reached the session once.
	if n := len(s.calls); n != 4 {
		t.Errorf("the session received %d statements, want 4", n)
	}
}

func TestPing(t *testing.T) {
	t.Parallel()
	s := &fakeSession{}
	if err := newTestDB(t, s).PingContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := s.lastCall(t).stmt; got != pingStmt {
		t.Errorf("got %q, want %q", got, pingStmt)
	}
}

func TestPrepare(t *testing.T) {
	t.Parallel()
	s := &fakeSession{result: result{cols: []gocql.ColumnInfo{column("n", native(gocql.TypeInt))}, rows: [][]any{{3}}}}
	db := newTestDB(t, s)
	st, err := db.PrepareContext(t.Context(), "SELECT n FROM t WHERE id = ?")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var n int
	if err := st.QueryRowContext(t.Context(), 7).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ExecContext(t.Context(), 8, PageSize(5)); err != nil {
		t.Fatal(err)
	}
	got := s.lastCall(t)
	if n != 3 || got.stmt != "SELECT n FROM t WHERE id = ?" || !reflect.DeepEqual(got.values, []any{8}) || *got.opts.pageSize != 5 {
		t.Errorf("got %d and %+v", n, got)
	}
}

func TestStmtWithNoContext(t *testing.T) {
	t.Parallel()
	st := &stmt{c: &conn{}, query: "SELECT 1"}
	if _, err := st.Exec(nil); !errors.Is(err, ErrNoContext) {
		t.Errorf("exec: got %v, want %v", err, ErrNoContext)
	}
	if _, err := st.Query(nil); !errors.Is(err, ErrNoContext) {
		t.Errorf("query: got %v, want %v", err, ErrNoContext)
	}
	if n := st.NumInput(); n != -1 {
		t.Errorf("got %d inputs, want -1", n)
	}
}

func TestNoTransactions(t *testing.T) {
	t.Parallel()
	db := newTestDB(t, &fakeSession{})
	if _, err := db.BeginTx(t.Context(), nil); !errors.Is(err, ErrNoTransactions) {
		t.Errorf("got %v, want %v", err, ErrNoTransactions)
	}
}
