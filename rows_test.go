package cql

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"math/big"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"gopkg.in/inf.v0"
)

// scanOne runs a query over a fake result of one column and one row, and
// scans the column into dest.
func scanOne(t *testing.T, info gocql.TypeInfo, value, dest any) error {
	t.Helper()
	s := &fakeSession{result: result{
		cols: []gocql.ColumnInfo{column("c", info)},
		rows: [][]any{{value}},
	}}
	return newTestDB(t, s).QueryRowContext(t.Context(), "SELECT c FROM t").Scan(dest)
}

func TestScanCanonical(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 9, 27, 10, 11, 12, 13_000_000, time.FixedZone("x", 3600))
	date := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	id := gocql.TimeUUID()
	varint, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
	tests := []struct {
		name  string
		info  gocql.TypeInfo
		value any
		want  any
	}{
		{"ascii", native(gocql.TypeAscii), "a", "a"},
		{"text", native(gocql.TypeText), "é", "é"},
		{"varchar", native(gocql.TypeVarchar), "v", "v"},
		{"blob", native(gocql.TypeBlob), []byte{1, 2}, []byte{1, 2}},
		{"empty blob", native(gocql.TypeBlob), empty{}, []byte{}},
		{"boolean", native(gocql.TypeBoolean), true, true},
		{"tinyint", native(gocql.TypeTinyInt), int8(-3), int64(-3)},
		{"smallint", native(gocql.TypeSmallInt), int16(300), int64(300)},
		{"int", native(gocql.TypeInt), 42, int64(42)},
		{"bigint", native(gocql.TypeBigInt), int64(1) << 40, int64(1) << 40},
		{"counter", native(gocql.TypeCounter), int64(7), int64(7)},
		{"float", native(gocql.TypeFloat), float32(1.5), float64(1.5)},
		{"double", native(gocql.TypeDouble), 2.25, 2.25},
		{"varint", native(gocql.TypeVarint), varint, "123456789012345678901234567890"},
		{"decimal", native(gocql.TypeDecimal), inf.NewDec(150, 2), "1.50"},
		{"timestamp", native(gocql.TypeTimestamp), ts, ts.UTC()},
		{"date", native(gocql.TypeDate), date, date},
		{"time", native(gocql.TypeTime), 90 * time.Second, 90 * time.Second},
		{"duration", native(gocql.TypeDuration), gocql.Duration{Months: 1, Days: 2, Nanoseconds: 3}, gocql.Duration{Months: 1, Days: 2, Nanoseconds: 3}},
		{"uuid", native(gocql.TypeUUID), id, uuid.UUID(id)},
		{"timeuuid", native(gocql.TypeTimeUUID), id, uuid.UUID(id)},
		{"inet", native(gocql.TypeInet), net.ParseIP("10.0.0.1"), "10.0.0.1"},
		{"list", parse("list<int>"), []int{1, 2}, []int{1, 2}},
		{"set", parse("set<text>"), []string{"a", "b"}, []string{"a", "b"}},
		{"map", parse("map<text, int>"), map[string]int{"a": 1}, map[string]int{"a": 1}},
		{"tuple", gocql.TupleTypeInfo{Elems: []gocql.TypeInfo{native(gocql.TypeInt), native(gocql.TypeText)}}, []any{7, "a"}, []any{int64(7), "a"}},
		{"tuple with null", gocql.TupleTypeInfo{Elems: []gocql.TypeInfo{native(gocql.TypeInt), native(gocql.TypeText)}}, []any{7, nil}, []any{int64(7), nil}},
		{"udt", udt(), map[string]any{"street": "main", "number": 4}, map[string]any{"street": "main", "number": 4}},
		{"vector", parse("vector<float, 3>"), []float32{1, 2, 3}, []float32{1, 2, 3}},
		{"empty int", native(gocql.TypeInt), empty{}, int64(0)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var got any
			if err := scanOne(t, test.info, test.value, &got); err != nil {
				t.Fatalf("scanning: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %#v (%T), want %#v (%T)", got, got, test.want, test.want)
			}
			if want := scanType(test.info); reflect.TypeOf(got) != want {
				t.Errorf("got type %T, and ColumnTypeScanType reports %v", got, want)
			}
		})
	}
}

// udt returns a user defined type with two fields.
func udt() gocql.UDTTypeInfo {
	return gocql.UDTTypeInfo{Keyspace: "ks", Name: "Address", Elements: []gocql.UDTField{
		{Name: "street", Type: native(gocql.TypeText)},
		{Name: "number", Type: native(gocql.TypeInt)},
	}}
}

func TestScanNull(t *testing.T) {
	t.Parallel()
	text := native(gocql.TypeText)
	tests := []struct {
		name string
		info gocql.TypeInfo
		dest any
		// want is the value that dest points to after the scan, or nil when
		// the scan must fail.
		want any
		// err is a part of the error, when the scan must fail.
		err string
	}{
		{"sql.Null", text, new(sql.Null[string]), sql.Null[string]{}, ""},
		{"sql.NullString", text, new(sql.NullString), sql.NullString{}, ""},
		{"any", text, new(any), nil, ""},
		{"pointer to pointer", text, new(*string), (*string)(nil), ""},
		{"slice", parse("list<text>"), &[]string{"old"}, []string(nil), ""},
		{"map", parse("map<text, int>"), &map[string]int{"old": 1}, map[string]int(nil), ""},
		{"string", text, new(string), nil, "converting NULL to string is unsupported"},
		{"int64", native(gocql.TypeBigInt), new(int64), nil, "converting NULL to int64 is unsupported"},
		{"time", native(gocql.TypeTimestamp), new(time.Time), nil, "unsupported Scan"},
		{"uuid", native(gocql.TypeUUID), new(gocql.UUID), nil, "unsupported Scan"},
		{"standard uuid", native(gocql.TypeUUID), new(uuid.UUID), nil, "unsupported Scan"},
		{"pointer to standard uuid", native(gocql.TypeUUID), new(*uuid.UUID), (*uuid.UUID)(nil), ""},
		{"sql.Null of standard uuid", native(gocql.TypeUUID), new(sql.Null[uuid.UUID]), sql.Null[uuid.UUID]{}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := scanOne(t, test.info, nil, test.dest)
			if test.err != "" {
				if err == nil || !strings.Contains(err.Error(), test.err) {
					t.Fatalf("got error %v, want one that holds %q", err, test.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("scanning: %v", err)
			}
			if got := reflect.ValueOf(test.dest).Elem().Interface(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestScanRawBytes(t *testing.T) {
	t.Parallel()
	s := &fakeSession{result: result{
		cols: []gocql.ColumnInfo{column("n", native(gocql.TypeInt)), column("s", native(gocql.TypeText))},
		rows: [][]any{{42, nil}},
	}}
	rows, err := newTestDB(t, s).QueryContext(t.Context(), "SELECT n, s FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("got no row: %v", rows.Err())
	}
	// sql.RawBytes holds the text form of the canonical value, never the wire
	// format.
	var n, str sql.RawBytes
	if err := rows.Scan(&n, &str); err != nil {
		t.Fatal(err)
	}
	if string(n) != "42" || str != nil {
		t.Errorf("got %q and %#v, want \"42\" and nil", n, str)
	}
}

// point decodes itself, so it decides what NULL is.
type point struct {
	null bool
	data string
}

func (p *point) UnmarshalCQL(_ gocql.TypeInfo, data []byte) error {
	p.null, p.data = data == nil, string(data)
	return nil
}

func TestScanDirect(t *testing.T) {
	t.Parallel()
	id := gocql.TimeUUID()
	ts := time.Date(2026, 9, 27, 10, 11, 12, 0, time.UTC)
	tests := []struct {
		name  string
		info  gocql.TypeInfo
		value any
		dest  any
		want  any
	}{
		{"list into slice", parse("list<text>"), []string{"a", "b"}, new([]string), []string{"a", "b"}},
		{"list into wider slice", parse("list<int>"), []int{1}, new([]int64), []int64{1}},
		{"map into map", parse("map<text, int>"), map[string]int{"a": 1}, new(map[string]int), map[string]int{"a": 1}},
		{"uuid into gocql uuid", native(gocql.TypeUUID), id, new(gocql.UUID), id},
		{"uuid into standard uuid", native(gocql.TypeUUID), id, new(uuid.UUID), uuid.UUID(id)},
		{"timeuuid into pointer to standard uuid", native(gocql.TypeTimeUUID), id, new(*uuid.UUID), new(uuid.UUID(id))},
		{"uuid into sql.Null of standard uuid", native(gocql.TypeUUID), id, new(sql.Null[uuid.UUID]), sql.Null[uuid.UUID]{V: uuid.UUID(id), Valid: true}},
		{"uuid into string", native(gocql.TypeUUID), id, new(string), id.String()},
		{"timestamp into time", native(gocql.TypeTimestamp), ts, new(time.Time), ts},
		{"int into int64", native(gocql.TypeInt), 42, new(int64), int64(42)},
		{"int into float64", native(gocql.TypeInt), 42, new(float64), float64(42)},
		{"bigint into string", native(gocql.TypeBigInt), int64(9), new(string), "9"},
		{"text into pointer", native(gocql.TypeText), "a", new(*string), new("a")},
		{"udt into map", udt(), map[string]any{"street": "main", "number": 4}, new(map[string]any), map[string]any{"street": "main", "number": 4}},
		{"text into unmarshaler", native(gocql.TypeText), "a", new(point), point{data: "a"}},
		{"null into unmarshaler", native(gocql.TypeText), nil, new(point), point{null: true}},
		{"empty into sql.Null", native(gocql.TypeInt), empty{}, new(sql.Null[int64]), sql.Null[int64]{Valid: true}},
		{"tuple into slice", gocql.TupleTypeInfo{Elems: []gocql.TypeInfo{native(gocql.TypeInt)}}, []any{5}, new([]any), []any{int64(5)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := scanOne(t, test.info, test.value, test.dest); err != nil {
				t.Fatalf("scanning: %v", err)
			}
			if got := reflect.ValueOf(test.dest).Elem().Interface(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestScanRefused(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		info  gocql.TypeInfo
		value any
		dest  any
	}{
		{"text into bool", native(gocql.TypeText), "a", new(bool)},
		{"list of text into slice of int", parse("list<text>"), []string{"a"}, new([]int)},
		{"bigint into int8", native(gocql.TypeBigInt), int64(1) << 40, new(int8)},
		{"not a pointer", native(gocql.TypeInt), 1, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := scanOne(t, test.info, test.value, test.dest); err == nil {
				t.Fatal("got no error")
			}
		})
	}
}

func TestScanManyRows(t *testing.T) {
	t.Parallel()
	s := &fakeSession{result: result{
		cols: []gocql.ColumnInfo{column("b", native(gocql.TypeBlob))},
		rows: [][]any{{[]byte("first")}, {[]byte("second")}},
	}}
	rows, err := newTestDB(t, s).QueryContext(t.Context(), "SELECT b FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	// Every []byte must keep its own bytes, although the capture buffer is
	// reused from row to row.
	var got [][]byte
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			t.Fatal(err)
		}
		got = append(got, b)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if want := [][]byte{[]byte("first"), []byte("second")}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRowsError(t *testing.T) {
	t.Parallel()
	fault := errors.New("read timeout")
	s := &fakeSession{result: result{
		cols: []gocql.ColumnInfo{column("c", native(gocql.TypeInt))},
		rows: [][]any{{1}},
		err:  fault,
	}}
	rows, err := newTestDB(t, s).QueryContext(t.Context(), "SELECT c FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	if n != 1 {
		t.Errorf("read %d rows, want 1", n)
	}
	if err := rows.Err(); !errors.Is(err, fault) {
		t.Errorf("got error %v, want %v", err, fault)
	}
}

func TestColumnTypes(t *testing.T) {
	t.Parallel()
	s := &fakeSession{result: result{cols: []gocql.ColumnInfo{
		column("a", native(gocql.TypeVarchar)),
		column("b", parse("frozen<map<text, list<int>>>")),
		column("c", udt()),
	}}}
	rows, err := newTestDB(t, s).QueryContext(t.Context(), "SELECT a, b, c FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	types, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name, db string
		scan     reflect.Type
	}{
		{"a", "VARCHAR", reflect.TypeFor[string]()},
		{"b", "MAP<TEXT, LIST<INT>>", reflect.TypeFor[map[string][]int]()},
		{"c", "Address", reflect.TypeFor[map[string]any]()},
	}
	for i, ct := range types {
		if ct.Name() != want[i].name || ct.DatabaseTypeName() != want[i].db || ct.ScanType() != want[i].scan {
			t.Errorf("column %d: got %s %s %v, want %s %s %v", i, ct.Name(), ct.DatabaseTypeName(), ct.ScanType(), want[i].name, want[i].db, want[i].scan)
		}
		if _, ok := ct.Nullable(); ok {
			t.Errorf("column %d: nullability is reported as known", i)
		}
	}
	if err := rows.Err(); err != nil {
		t.Error(err)
	}
}

func TestTypeName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		info gocql.TypeInfo
		want string
	}{
		{native(gocql.TypeText), "TEXT"},
		{native(gocql.TypeTimeUUID), "TIMEUUID"},
		{parse("list<int>"), "LIST<INT>"},
		{parse("set<uuid>"), "SET<UUID>"},
		{parse("map<text, frozen<list<int>>>"), "MAP<TEXT, LIST<INT>>"},
		{gocql.TupleTypeInfo{Elems: []gocql.TypeInfo{native(gocql.TypeInt), native(gocql.TypeText)}}, "TUPLE<INT, TEXT>"},
		{parse("vector<float, 3>"), "VECTOR<FLOAT, 3>"},
		{udt(), "Address"},
	}
	for _, test := range tests {
		if got := typeName(test.info); got != test.want {
			t.Errorf("got %s, want %s", got, test.want)
		}
	}
}

func TestNext(t *testing.T) {
	t.Parallel()
	it := &fakeIter{res: result{
		cols: []gocql.ColumnInfo{column("a", native(gocql.TypeInt)), column("b", native(gocql.TypeText))},
		rows: [][]any{{1, nil}},
	}}
	r := newRows(it)
	dest := make([]driver.Value, 2)
	if err := r.Next(dest); err != nil {
		t.Fatal(err)
	}
	if want := []driver.Value{int64(1), nil}; !reflect.DeepEqual(dest, want) {
		t.Errorf("got %#v, want %#v", dest, want)
	}
	if err := r.Next(dest); !errors.Is(err, io.EOF) {
		t.Errorf("got %v, want io.EOF", err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("closing rows: %v", err)
	}
}

func BenchmarkScan(b *testing.B) {
	it := &fakeIter{}
	res := result{cols: []gocql.ColumnInfo{
		column("id", native(gocql.TypeUUID)),
		column("name", native(gocql.TypeText)),
		column("n", native(gocql.TypeBigInt)),
	}, rows: [][]any{{gocql.TimeUUID(), "name", int64(1)}}}
	var id gocql.UUID
	var name string
	var n int64
	b.ReportAllocs()
	for b.Loop() {
		it.res, it.pos = res, 0
		r := newRows(it)
		if err := r.NextRow(); err != nil {
			b.Fatal(err)
		}
		for i, dest := range []any{&id, &name, &n} {
			if err := r.ScanColumn(driver.ScanContext{}, i, dest); err != nil {
				b.Fatal(err)
			}
		}
	}
}
