package cassandra_test

import (
	"fmt"
	"math/big"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/cockroachdb/apd/v3"
	"github.com/xo/dbimp"
	"github.com/xo/dbimp/dbimptest"
)

// rtCase is the round trip of one type: the type of the column, the values to
// write, and how to write one as a CQL literal.
type rtCase struct {
	column  string
	values  []dbimptest.Value
	literal func(any) (string, error)
	equal   func(got, want any) bool
}

// text writes a CQL string literal.
func text(v any) (string, error) {
	return "'" + strings.ReplaceAll(fmt.Sprint(v), "'", "''") + "'", nil
}

// literalOf adapts a function of the Go type of the values of a case to the
// function that writes a literal. It returns an error for any other type.
func literalOf[T any](f func(T) (string, error)) func(any) (string, error) {
	return func(v any) (string, error) {
		t, ok := v.(T)
		if !ok {
			return "", fmt.Errorf("a literal for %T: %w", v, dbimp.ErrNotSupported)
		}
		return f(t)
	}
}

// plain writes a value with fmt.Sprint, which is a CQL literal for a number or
// a boolean.
func plain(v any) (string, error) { return fmt.Sprint(v), nil }

// quoted writes the text of a value in single quotes.
func quoted(v any) (string, error) { return "'" + fmt.Sprint(v) + "'", nil }

// orNil wraps a comparison so that nil equals nil, and an empty collection
// equals NULL, because Cassandra stores an empty collection as NULL.
func orNil(equal func(got, want any) bool) func(got, want any) bool {
	return func(got, want any) bool {
		if got == nil || want == nil {
			return got == nil && (want == nil || isEmpty(want))
		}
		if equal == nil {
			return reflect.DeepEqual(got, want)
		}
		return equal(got, want)
	}
}

// isEmpty reports whether v is an empty slice or map.
func isEmpty(v any) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	}
	return false
}

// bigEqual compares two integers of any size.
func bigEqual(got, want any) bool {
	g, gok := got.(*big.Int)
	w, wok := want.(*big.Int)
	return gok && wok && g.Cmp(w) == 0
}

// decimalEqual compares two decimals by their value.
func decimalEqual(got, want any) bool {
	g, gok := got.(*apd.Decimal)
	w, wok := want.(*apd.Decimal)
	return gok && wok && g.Cmp(w) == 0
}

// timeEqual compares two instants.
func timeEqual(got, want any) bool {
	g, gok := got.(time.Time)
	w, wok := want.(time.Time)
	return gok && wok && g.Equal(w)
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("not an integer: " + s)
	}
	return v
}

func mustDecimal(s string) *apd.Decimal {
	v, _, err := apd.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return v
}

// ints returns the case of an integer column, from its smallest value to its
// largest.
func ints[T int8 | int16 | int32 | int64](column string, lo, hi T) rtCase {
	return rtCase{column: column, literal: plain, values: []dbimptest.Value{
		{Name: "min", In: lo, Want: int64(lo)},
		{Name: "zero", In: T(0), Want: int64(0)},
		{Name: "max", In: hi, Want: int64(hi)},
		{Name: "null"},
	}}
}

// rtCases are the round trips, by the name of the type.
func rtCases() map[string]rtCase {
	id, tu := uuid.NewV7(), gocql.TimeUUID()
	ts := time.Date(2026, 9, 27, 10, 11, 12, 13_000_000, time.UTC)
	return map[string]rtCase{
		"ascii": {column: "ascii", literal: text, values: []dbimptest.Value{
			{Name: "empty", In: ""}, {Name: "text", In: "a'b"}, {Name: "long", In: strings.Repeat("x", 4096)}, {Name: "null"},
		}},
		"text": {column: "text", literal: text, values: []dbimptest.Value{
			{Name: "empty", In: ""}, {Name: "unicode", In: "é日本語 🙂"}, {Name: "long", In: strings.Repeat("é", 4096)}, {Name: "null"},
		}},
		"varchar": {column: "varchar", literal: text, values: []dbimptest.Value{
			{Name: "empty", In: ""}, {Name: "unicode", In: "é日本語"}, {Name: "null"},
		}},
		"boolean": {column: "boolean", literal: plain, values: []dbimptest.Value{
			{Name: "true", In: true}, {Name: "false", In: false}, {Name: "null"},
		}},
		"tinyint":  ints[int8]("tinyint", -128, 127),
		"smallint": ints[int16]("smallint", -32768, 32767),
		"int":      ints[int32]("int", -2147483648, 2147483647),
		"bigint":   ints[int64]("bigint", -9223372036854775808, 9223372036854775807),
		"float": {column: "float", literal: plain, values: []dbimptest.Value{
			{Name: "min", In: float32(-3.4028235e38), Want: float64(float32(-3.4028235e38))},
			{Name: "zero", In: float32(0), Want: float64(0)},
			{Name: "fraction", In: float32(1.5), Want: float64(1.5)},
			{Name: "null"},
		}},
		"double": {column: "double", literal: plain, values: []dbimptest.Value{
			{Name: "min", In: -1.7976931348623157e308}, {Name: "zero", In: float64(0)}, {Name: "fraction", In: 2.25}, {Name: "null"},
		}},
		"decimal": {column: "decimal", equal: decimalEqual, literal: plain, values: []dbimptest.Value{
			{Name: "zero", In: mustDecimal("0")},
			{Name: "fraction", In: mustDecimal("1.50")},
			{Name: "negative", In: mustDecimal("-123456789.987654321")},
			{Name: "large", In: mustDecimal("12345678901234567890.123456789012345678901234567890")},
			{Name: "null"},
		}},
		"varint": {column: "varint", equal: bigEqual, literal: plain, values: []dbimptest.Value{
			{Name: "zero", In: big.NewInt(0)},
			{Name: "negative", In: mustBig("-123456789012345678901234567890")},
			{Name: "large", In: mustBig("123456789012345678901234567890123456789012345678901234567890")},
			{Name: "null"},
		}},
		"blob": {column: "blob", literal: func(v any) (string, error) { return fmt.Sprintf("0x%x", v), nil }, values: []dbimptest.Value{
			{Name: "empty", In: []byte{}}, {Name: "bytes", In: []byte{0, 1, 2, 255}}, {Name: "long", In: make([]byte, 4096)}, {Name: "null"},
		}},
		"uuid": {column: "uuid", literal: plain, values: []dbimptest.Value{
			{Name: "v7", In: id}, {Name: "nil", In: uuid.UUID{}}, {Name: "null"},
		}},
		"timeuuid": {column: "timeuuid", literal: plain, values: []dbimptest.Value{
			{Name: "time", In: uuid.UUID(tu)}, {Name: "null"},
		}},
		"inet": {column: "inet", literal: quoted, values: []dbimptest.Value{
			{Name: "v4", In: netip.MustParseAddr("10.0.0.1")}, {Name: "v6", In: netip.MustParseAddr("2001:db8::1")}, {Name: "loopback", In: netip.MustParseAddr("::1")}, {Name: "null"},
		}},
		"timestamp": {column: "timestamp", equal: timeEqual, literal: func(v any) (string, error) {
			t, ok := v.(time.Time)
			if !ok {
				return "", fmt.Errorf("a literal for %T: %w", v, dbimp.ErrNotSupported)
			}
			return "'" + t.UTC().Format("2006-01-02T15:04:05.000Z") + "'", nil
		}, values: []dbimptest.Value{
			{Name: "time", In: ts}, {Name: "epoch", In: time.UnixMilli(0).UTC()}, {Name: "null"},
		}},
		"date": {column: "date", literal: quoted, values: []dbimptest.Value{
			{Name: "date", In: dbimp.Date{Year: 2026, Month: time.September, Day: 27}},
			{Name: "epoch", In: dbimp.Date{Year: 1970, Month: time.January, Day: 1}},
			{Name: "old", In: dbimp.Date{Year: 1, Month: time.January, Day: 1}},
			{Name: "null"},
		}},
		"time": {column: "time", literal: quoted, values: []dbimptest.Value{
			{Name: "midnight", In: dbimp.LocalTime{}},
			{Name: "nanoseconds", In: dbimp.LocalTime{Hour: 12, Minute: 30, Second: 45, Nanosecond: 123456789}},
			{Name: "last", In: dbimp.LocalTime{Hour: 23, Minute: 59, Second: 59, Nanosecond: 999999999}},
			{Name: "null"},
		}},
		"duration": {column: "duration", literal: literalOf(func(iv dbimp.Interval) (string, error) {
			if iv.Months < 0 || iv.Days < 0 || iv.Nanoseconds < 0 {
				return "", fmt.Errorf("a literal for a negative duration: %w", dbimp.ErrNotSupported)
			}
			return fmt.Sprintf("%dmo%dd%dns", iv.Months, iv.Days, iv.Nanoseconds), nil
		}), values: []dbimptest.Value{
			{Name: "zero", In: dbimp.Interval{}},
			{Name: "parts", In: dbimp.Interval{Months: 1, Days: 2, Nanoseconds: 3}},
			{Name: "negative", In: dbimp.Interval{Months: -1, Days: -2, Nanoseconds: -3}},
			{Name: "null"},
		}},
		"list": {column: "list<int>", literal: func(v any) (string, error) {
			return fmt.Sprintf("[%s]", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(v)), ", "), "[]")), nil
		}, values: []dbimptest.Value{
			{Name: "ints", In: []int32{1, 2, 3}, Want: []any{int64(1), int64(2), int64(3)}},
			{Name: "one", In: []int32{7}, Want: []any{int64(7)}},
			{Name: "empty", In: []int32{}},
			{Name: "null"},
		}},
		"set": {column: "set<text>", literal: literalOf(func(v []string) (string, error) {
			if len(v) == 0 {
				return "{}", nil
			}
			return "{'" + strings.Join(v, "', '") + "'}", nil
		}), values: []dbimptest.Value{
			{Name: "texts", In: []string{"a", "b"}, Want: []any{"a", "b"}},
			{Name: "empty", In: []string{}},
			{Name: "null"},
		}},
		"map": {column: "map<text, int>", literal: literalOf(func(v map[string]int32) (string, error) {
			var pairs []string
			for k, n := range v {
				pairs = append(pairs, fmt.Sprintf("'%s': %d", k, n))
			}
			return "{" + strings.Join(pairs, ", ") + "}", nil
		}), values: []dbimptest.Value{
			{Name: "pairs", In: map[string]int32{"a": 1, "b": 2}, Want: map[string]any{"a": int64(1), "b": int64(2)}},
			{Name: "empty", In: map[string]int32{}},
			{Name: "null"},
		}},
		"vector": {column: "vector<float, 3>", literal: literalOf(func(vec dbimp.Vector[float32]) (string, error) {
			return fmt.Sprintf("[%g, %g, %g]", vec[0], vec[1], vec[2]), nil
		}), values: []dbimptest.Value{
			{Name: "floats", In: dbimp.Vector[float32]{1, 2.5, -3}},
			{Name: "zeros", In: dbimp.Vector[float32]{0, 0, 0}},
			{Name: "null"},
		}},
	}
}

// TestIntegrationRoundTrip stores values of each type in a column, and reads
// them back through Rows.Scan, as step 14a of the driver guide of dbimp
// requires. It runs against the server that CASSANDRA_DSN names.
//
// A counter, a tuple and a user defined type are not here. A counter column
// takes no INSERT. gocql v2.1.2 cannot bind a tuple (see TestIntegrationTypes),
// and a user defined type needs a type that the test makes first.
func TestIntegrationRoundTrip(t *testing.T) {
	db, ks := openTestDB(t)
	for name, c := range rtCases() {
		t.Run(name, func(t *testing.T) {
			table := ks + ".rt_" + name
			// A release that has no vector type refuses the table, and the
			// round trip skips it with the reason.
			if strings.HasPrefix(c.column, "vector") {
				probe := ks + ".rt_probe"
				if _, err := db.ExecContext(t.Context(), "CREATE TABLE "+probe+" (k text PRIMARY KEY, v "+c.column+")"); err != nil {
					t.Skipf("the server has no %s: %v", c.column, err)
				}
				exec(t, db, "DROP TABLE "+probe)
			}
			rt := dbimptest.RoundTripCase{
				Type:     name,
				Setup:    []string{"CREATE TABLE " + table + " (k text PRIMARY KEY, v " + c.column + ")"},
				Teardown: []string{"DROP TABLE " + table},
				Insert:   "INSERT INTO " + table + " (k, v) VALUES (?, ?)",
				Select:   "SELECT v FROM " + table + " WHERE k = ?",
				Update:   "UPDATE " + table + " SET v = ? WHERE k = ?",
				Delete:   "DELETE FROM " + table + " WHERE k = ?",
				Values:   c.values,
				Equal:    orNil(c.equal),
				Literal: func(key string, v any) (string, error) {
					if c.literal == nil {
						return "", fmt.Errorf("a literal for %s: %w", name, dbimp.ErrNotSupported)
					}
					if v == nil {
						return "INSERT INTO " + table + " (k, v) VALUES ('" + key + "', null)", nil
					}
					lit, err := c.literal(v)
					if err != nil {
						return "", err
					}
					return "INSERT INTO " + table + " (k, v) VALUES ('" + key + "', " + lit + ")", nil
				},
			}
			dbimptest.RoundTrip(t, db, rt)
		})
	}
}
