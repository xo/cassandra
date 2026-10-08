package cassandra

import (
	"encoding/hex"
	"errors"
	"math/big"
	"net"
	"net/netip"
	"reflect"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/cockroachdb/apd/v3"
	"github.com/xo/dbimp"
)

// TestDecimalWire holds the wire form of a decimal: the scale as an int32, then
// the unscaled value as the shortest big endian two's complement integer. The
// expected bytes are what Java BigDecimal writes.
func TestDecimalWire(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		in   string
		want string
	}{
		{"0", "00000000" + "00"},
		{"1.50", "00000002" + "0096"},
		{"-1.50", "00000002" + "ff6a"},
		{"127", "00000000" + "7f"},
		{"128", "00000000" + "0080"},
		{"-128", "00000000" + "80"},
		{"-129", "00000000" + "ff7f"},
		{"1E+3", "fffffffd" + "01"},
		{"12345678901234567890.123456789", ""},
	} {
		d, _, err := apd.NewFromString(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decimal{d}.MarshalCQL(nil)
		if err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		// A value with no expected bytes is only read back.
		if tt.want != "" && hex.EncodeToString(got) != tt.want {
			t.Errorf("%s: got %x, want %s", tt.in, got, tt.want)
		}
		// Reading the bytes back through gocql gives the same number.
		info := native(gocql.TypeDecimal)
		back, err := canonical(info, false, got)
		if err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		if got, ok := back.(*apd.Decimal); !ok || got.Cmp(d) != 0 {
			t.Errorf("%s: read back %s", tt.in, back)
		}
	}
}

// TestDecimalRefusesWhatIsNotANumber holds that a NaN and an infinity are
// errors, because Cassandra has no form of either.
func TestDecimalRefusesWhatIsNotANumber(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"NaN", "Infinity", "-Infinity"} {
		d, _, err := apd.NewFromString(s)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := (decimal{d}).MarshalCQL(nil); !errors.Is(err, dbimp.ErrInvalidValue) {
			t.Errorf("%s: got %v, want %v", s, err, dbimp.ErrInvalidValue)
		}
	}
}

// TestBind holds that each canonical type is also a type that a caller can
// bind, and that bind turns it into what gocql takes.
func TestBind(t *testing.T) {
	t.Parallel()
	dec := apd.New(150, -2)
	for _, tt := range []struct {
		name string
		in   any
		want any
	}{
		{"time", dbimp.LocalTime{Hour: 1, Minute: 2, Second: 3, Nanosecond: 4}, time.Hour + 2*time.Minute + 3*time.Second + 4},
		{"interval", dbimp.Interval{Months: 1, Days: 2, Nanoseconds: 3}, gocql.Duration{Months: 1, Days: 2, Nanoseconds: 3}},
		{"ipv4", netip.MustParseAddr("10.0.0.1"), net.IP{10, 0, 0, 1}},
		{"decimal", dec, decimal{dec}},
		{"vector", dbimp.Vector[float32]{1, 2}, []float32{1, 2}},
		{"nil decimal", (*apd.Decimal)(nil), nil},
	} {
		got, ok := bind(tt.in)
		if !ok || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %#v %v, want %#v", tt.name, got, ok, tt.want)
		}
	}
	if _, ok := bind("text"); ok {
		t.Error("bind took a string, which gocql takes as it is")
	}
}

// TestInetIsAnAddress holds that an IPv4 address and an IPv6 address are a
// netip.Addr, and that an IPv4 address is not the form that is mapped into 16
// bytes.
func TestInetIsAnAddress(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"10.0.0.1", "::1", "2001:db8::1"} {
		got, err := canonical(native(gocql.TypeInet), false, net.ParseIP(s))
		if err != nil {
			t.Fatal(err)
		}
		if want := netip.MustParseAddr(s); got != want {
			t.Errorf("%s: got %v, want %v", s, got, want)
		}
	}
}

// TestVarintIsABigInteger holds that a varint is a *big.Int, whatever its size.
func TestVarintIsABigInteger(t *testing.T) {
	t.Parallel()
	want, _ := new(big.Int).SetString("-123456789012345678901234567890", 10)
	data, err := gocql.Marshal(native(gocql.TypeVarint), want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := canonical(native(gocql.TypeVarint), false, data)
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := got.(*big.Int); !ok || b.Cmp(want) != 0 {
		t.Errorf("got %v (%T), want %v", got, got, want)
	}
}

// TestDateWire holds the wire form of a date: the days since 1970-01-01 plus
// 2**31. The day 0001-01-01 is the zero time.Time, which gocql writes as an
// empty value, so the driver writes the date itself.
func TestDateWire(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		date dbimp.Date
		want string
	}{
		{dbimp.Date{Year: 1970, Month: time.January, Day: 1}, "80000000"},
		{dbimp.Date{Year: 1970, Month: time.January, Day: 2}, "80000001"},
		{dbimp.Date{Year: 1969, Month: time.December, Day: 31}, "7fffffff"},
		{dbimp.Date{Year: 1, Month: time.January, Day: 1}, "7ff506c6"},
		{dbimp.Date{Year: 2026, Month: time.September, Day: 27}, "800050f3"},
	} {
		got, ok := bind(tt.date)
		if !ok {
			t.Fatalf("%v: bind took no date", tt.date)
		}
		m, ok := got.(gocql.Marshaler)
		if !ok {
			t.Fatalf("%v: bind took %T, which is not a gocql.Marshaler", tt.date, got)
		}
		data, err := m.MarshalCQL(nil)
		if err != nil || hex.EncodeToString(data) != tt.want {
			t.Errorf("%v: got %x and %v, want %s", tt.date, data, err, tt.want)
		}
		// gocql reads the bytes back as the same day.
		back, err := canonical(native(gocql.TypeDate), false, data)
		if err != nil || back != tt.date {
			t.Errorf("%v: read back %v and %v", tt.date, back, err)
		}
	}
	if _, err := (date{dbimp.Date{Year: 9000000, Month: time.January, Day: 1}}).MarshalCQL(nil); !errors.Is(err, dbimp.ErrInvalidValue) {
		t.Errorf("got %v, want %v for a date outside the range", err, dbimp.ErrInvalidValue)
	}
}
