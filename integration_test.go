package cassandra_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/cockroachdb/apd/v3"
	"github.com/xo/cassandra"
	"github.com/xo/dbimp"
)

// openTestDB opens the server that CASSANDRA_DSN names, and creates a keyspace for
// the test. The test drops the keyspace when it ends. The test is skipped
// when CASSANDRA_DSN is empty. Start a server with dbrun from dbmeta:
//
//	cd ../dbmeta/test && go run ./cmd/dbrun start cassandra-5.0
//	export CASSANDRA_DSN=$(go run ./cmd/dbrun dsn cassandra-5.0)
func openTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("CASSANDRA_DSN")
	if dsn == "" {
		t.Skip("CASSANDRA_DSN is empty")
	}
	db, err := sql.Open("cassandra", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing db: %v", err)
		}
	})
	// NetworkTopologyStrategy in the data center of the node works on every
	// release. ScyllaDB with tablets refuses SimpleStrategy.
	var dc string
	if err := db.QueryRowContext(t.Context(), "SELECT data_center FROM system.local").Scan(&dc); err != nil {
		t.Fatalf("reading the data center: %v", err)
	}
	keyspace := fmt.Sprintf("cassandratest_%d", time.Now().UnixNano())
	exec(t, db, "CREATE KEYSPACE "+keyspace+" WITH replication = {'class': 'NetworkTopologyStrategy', '"+dc+"': 1}")
	t.Cleanup(func() {
		// The test context is done by now.
		if _, err := db.ExecContext(context.WithoutCancel(t.Context()), "DROP KEYSPACE "+keyspace); err != nil {
			t.Errorf("dropping keyspace %s: %v", keyspace, err)
		}
	})
	return db, keyspace
}

func exec(t *testing.T, db *sql.DB, stmt string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(), stmt, args...); err != nil {
		t.Fatalf("executing %s: %v", stmt, err)
	}
}

func TestIntegrationTypes(t *testing.T) {
	db, ks := openTestDB(t)
	exec(t, db, "CREATE TYPE "+ks+".address (street text, number int)")
	exec(t, db, "CREATE TABLE "+ks+`.everything (
		id uuid PRIMARY KEY, a ascii, t text, b blob, bo boolean, ti tinyint, si smallint,
		i int, bi bigint, f float, d double, vi varint, de decimal, ts timestamp,
		da date, tm time, du duration, tu timeuuid, ip inet, l list<int>, s set<text>,
		m map<text, int>, tp tuple<int, text>, ad frozen<address>)`)
	id := gocql.TimeUUID()
	tu := gocql.TimeUUID()
	ts := time.Date(2026, 9, 27, 10, 11, 12, 13_000_000, time.UTC)
	date := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	vi, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
	// gocql v2.1.2 cannot bind a tuple: it counts each element of a tuple
	// marker as a value of its own. So the tuple is a literal here.
	exec(t, db, "INSERT INTO "+ks+`.everything (id, a, t, b, bo, ti, si, i, bi, f, d, vi, de, ts, da, tm, du, tu, ip, l, s, m, tp, ad)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, (7, 'seven'), ?)`,
		id, "a", "é", []byte{1, 2}, true, int8(-3), int16(300), 42, int64(1)<<40, float32(1.5), 2.25,
		vi, apd.New(150, -2), ts, date, 90*time.Second, gocql.Duration{Months: 1, Days: 2, Nanoseconds: 3},
		tu, net.ParseIP("10.0.0.1"), []int{1, 2}, []string{"x", "y"}, map[string]int{"k": 1},
		map[string]any{"street": "main", "number": 4})
	// Each value has the Go type of its CQL type (dbimp D135).
	want := map[string]any{
		"id": uuid.UUID(id), "a": "a", "t": "é", "b": []byte{1, 2}, "bo": true,
		"ti": int64(-3), "si": int64(300), "i": int64(42), "bi": int64(1) << 40,
		"f": float64(1.5), "d": 2.25, "vi": vi, "de": apd.New(150, -2), "ts": ts,
		"da": dbimp.Date{Year: 2026, Month: time.September, Day: 27},
		"tm": dbimp.LocalTime{Minute: 1, Second: 30},
		"du": dbimp.Interval{Months: 1, Days: 2, Nanoseconds: 3},
		"tu": uuid.UUID(tu), "ip": netip.MustParseAddr("10.0.0.1"),
		"l": []any{int64(1), int64(2)}, "s": []any{"x", "y"},
		"m": map[string]any{"k": int64(1)}, "tp": []any{int64(7), "seven"},
		"ad": map[string]any{"street": "main", "number": int64(4)},
	}
	rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+ks+".everything WHERE id = ?", id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}
	if !rows.Next() {
		t.Fatalf("got no row: %v", rows.Err())
	}
	values := make([]any, len(names))
	dest := make([]any, len(names))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := rows.Scan(dest...); err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		if !sameValue(values[i], want[name]) {
			t.Errorf("%s: got %#v (%T), want %#v", name, values[i], values[i], want[name])
		}
		if st := types[i].ScanType(); reflect.TypeOf(values[i]) != st {
			t.Errorf("%s: got %T, and the scan type is %v", name, values[i], st)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	var tags []string
	var u gocql.UUID
	if err := db.QueryRowContext(t.Context(), "SELECT s, id FROM "+ks+".everything WHERE id = ?", id).Scan(&tags, &u); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tags, []string{"x", "y"}) || u != id {
		t.Errorf("got %v and %v", tags, u)
	}
	// The standard uuid.UUID binds and scans (D25).
	var std uuid.UUID
	var null sql.Null[uuid.UUID]
	q := "SELECT id, tu FROM " + ks + ".everything WHERE id = ?"
	if err := db.QueryRowContext(t.Context(), q, uuid.UUID(id)).Scan(&std, &null); err != nil {
		t.Fatal(err)
	}
	if std != uuid.UUID(id) || !null.Valid || null.V != uuid.UUID(tu) {
		t.Errorf("got %v and %v", std, null)
	}
}

// sameValue compares two values of a row. A *big.Int and an *apd.Decimal hold
// their number in a form that reflect.DeepEqual cannot compare.
func sameValue(got, want any) bool {
	switch w := want.(type) {
	case *big.Int:
		g, ok := got.(*big.Int)
		return ok && g.Cmp(w) == 0
	case *apd.Decimal:
		g, ok := got.(*apd.Decimal)
		return ok && g.Cmp(w) == 0
	}
	return reflect.DeepEqual(got, want)
}

func TestIntegrationNull(t *testing.T) {
	db, ks := openTestDB(t)
	exec(t, db, "CREATE TABLE "+ks+".n (id int PRIMARY KEY, t text, l list<text>)")
	exec(t, db, "INSERT INTO "+ks+".n (id, l) VALUES (?, ?)", 1, []string{})
	var (
		t1 sql.Null[string]
		t2 *string
		l  []string
		a  any
	)
	q := "SELECT t, t, l, t FROM " + ks + ".n WHERE id = 1"
	if err := db.QueryRowContext(t.Context(), q).Scan(&t1, &t2, &l, &a); err != nil {
		t.Fatal(err)
	}
	if t1.Valid || t2 != nil || l != nil || a != nil {
		t.Errorf("got %v, %v, %v and %v, want every one NULL", t1, t2, l, a)
	}
	var s string
	err := db.QueryRowContext(t.Context(), "SELECT t FROM "+ks+".n WHERE id = 1").Scan(&s)
	if err == nil || !strings.Contains(err.Error(), "converting NULL to string is unsupported") {
		t.Errorf("got %v, want the error for NULL into a string", err)
	}
	// ScyllaDB refuses the type hint (text)NULL, so the check runs only where
	// the server accepts it.
	var pad sql.Null[string]
	var syntax *gocql.RequestErrSyntax
	switch err := db.QueryRowContext(t.Context(), "SELECT (text)NULL FROM "+ks+".n WHERE id = 1").Scan(&pad); {
	case errors.As(err, &syntax):
		t.Log("the server refuses (text)NULL")
	case err != nil || pad.Valid:
		t.Errorf("got %v and %v, want an invalid sql.Null (dbmeta D62)", pad, err)
	}
}

func TestIntegrationErrors(t *testing.T) {
	db, ks := openTestDB(t)
	var syntax *gocql.RequestErrSyntax
	if _, err := db.QueryContext(t.Context(), "SELECT *FROMM "+ks+".nope"); !errors.As(err, &syntax) {
		t.Errorf("syntax error: got %v (%T)", err, err)
	}
	var invalid *gocql.RequestErrInvalid
	if _, err := db.QueryContext(t.Context(), "SELECT * FROM "+ks+".nope"); !errors.As(err, &invalid) {
		t.Errorf("missing table: got %v (%T)", err, err)
	}
	cfg, err := cassandra.ParseDSN(os.Getenv("CASSANDRA_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Keyspace = "no_such_keyspace"
	if err := sql.OpenDB(cassandra.NewConnector(cfg)).PingContext(t.Context()); err == nil {
		t.Error("a missing keyspace gave no error")
	}
	auth, ok := cfg.Authenticator.(gocql.PasswordAuthenticator)
	if !ok {
		t.Skip("CASSANDRA_DSN has no credentials, so the wrong password case cannot run")
	}
	cfg.Keyspace = ""
	auth.Password += "-wrong"
	cfg.Authenticator = auth
	err = sql.OpenDB(cassandra.NewConnector(cfg)).PingContext(t.Context())
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "password") {
		t.Errorf("wrong password: got %v, want an error that names the password", err)
	}
}

func TestIntegrationPaging(t *testing.T) {
	db, ks := openTestDB(t)
	exec(t, db, "CREATE TABLE "+ks+".p (k int, c int, PRIMARY KEY (k, c))")
	for i := range 25 {
		exec(t, db, "INSERT INTO "+ks+".p (k, c) VALUES (1, ?)", i)
	}
	ctx := cassandra.WithOptions(t.Context(), cassandra.WithConsistency(gocql.One))
	rows, err := db.QueryContext(ctx, "SELECT c FROM "+ks+".p WHERE k = 1", cassandra.WithPageSize(10))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var c int
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		if c != n {
			t.Errorf("row %d holds %d", n, c)
		}
		n++
	}
	if err := rows.Err(); err != nil || n != 25 {
		t.Errorf("read %d rows and %v, want 25 and no error", n, err)
	}
}

func TestIntegrationCancel(t *testing.T) {
	db, _ := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := db.ExecContext(ctx, "SELECT * FROM system.local"); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}
}

func TestIntegrationBatchAndLightweightTransaction(t *testing.T) {
	db, ks := openTestDB(t)
	exec(t, db, "CREATE TABLE "+ks+".u (id int PRIMARY KEY, name text)")
	exec(t, db, "BEGIN BATCH INSERT INTO "+ks+".u (id, name) VALUES (?, ?); INSERT INTO "+ks+".u (id, name) VALUES (?, ?); APPLY BATCH",
		1, "one", 2, "two")
	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+ks+".u").Scan(&n); err != nil || n != 2 {
		t.Fatalf("got %d rows and %v, want 2", n, err)
	}
	for _, want := range []bool{true, false} {
		rows, err := db.QueryContext(t.Context(), "INSERT INTO "+ks+".u (id, name) VALUES (3, 'three') IF NOT EXISTS")
		if err != nil && strings.Contains(err.Error(), "not yet supported with tablets") {
			t.Skipf("the server refuses a lightweight transaction: %v", err)
		}
		if err != nil {
			t.Fatal(err)
		}
		if !rows.Next() {
			t.Fatalf("got no row: %v", rows.Err())
		}
		// The first column is [applied]. When it is false, the row also holds
		// the values that are there.
		cols, _ := rows.Columns()
		dest := make([]any, len(cols))
		var applied bool
		dest[0] = &applied
		for i := 1; i < len(dest); i++ {
			dest[i] = new(any)
		}
		if err := rows.Scan(dest...); err != nil {
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		if applied != want {
			t.Errorf("got applied %v, want %v", applied, want)
		}
	}
	if _, err := db.BeginTx(t.Context(), nil); !errors.Is(err, dbimp.ErrNotSupported) {
		t.Errorf("got %v, want %v", err, dbimp.ErrNotSupported)
	}
}

// TestIntegrationOrdinary runs as the ordinary user that dbmeta makes, which
// can read the system tables and nothing else. It is skipped when
// CASSANDRA_ORDINARY_DSN is empty.
func TestIntegrationOrdinary(t *testing.T) {
	db, ks := openTestDB(t)
	dsn := os.Getenv("CASSANDRA_ORDINARY_DSN")
	if dsn == "" {
		t.Skip("CASSANDRA_ORDINARY_DSN is empty")
	}
	exec(t, db, "CREATE TABLE "+ks+".o (id int PRIMARY KEY, v text)")
	exec(t, db, "INSERT INTO "+ks+".o (id, v) VALUES (1, 'a')")
	user, err := sql.Open("cassandra", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer user.Close()
	if err := user.PingContext(t.Context()); err != nil {
		t.Fatalf("pinging as the ordinary user: %v", err)
	}
	var version string
	if err := user.QueryRowContext(t.Context(), "SELECT release_version FROM system.local").Scan(&version); err != nil || version == "" {
		t.Errorf("reading the version as the ordinary user: %q, %v", version, err)
	}
	// The ordinary user has no permission on the table of the test, and the
	// error reaches the caller as the error of gocql.
	_, err = user.QueryContext(t.Context(), "SELECT v FROM "+ks+".o WHERE id = 1")
	if _, ok := errors.AsType[*gocql.RequestErrUnauthorized](err); !ok {
		t.Errorf("reading a table with no permission: got %v (%T), want *gocql.RequestErrUnauthorized", err, err)
	}
}
