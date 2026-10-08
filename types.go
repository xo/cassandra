package cassandra

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"time"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/cockroachdb/apd/v3"
	"github.com/xo/dbimp"
)

// The canonical value of a column is what *any receives, what a sql.Scanner
// receives, what Next returns, and what ColumnTypeScanType reports. One CQL
// type has one canonical Go type in all four places, by the kinds of the types
// document of dbimp (dbimp D135):
//
//	ascii, text, varchar            string
//	boolean                         bool
//	tinyint, smallint, int, bigint  int64
//	counter                         int64
//	float, double                   float64
//	decimal                         *apd.Decimal
//	varint                          *big.Int
//	blob                            []byte
//	uuid, timeuuid                  uuid.UUID
//	inet                            netip.Addr
//	timestamp                       time.Time
//	date                            dbimp.Date
//	time                            dbimp.LocalTime
//	duration                        dbimp.Interval
//	list, set, tuple                []any
//	map with a text key, UDT        map[string]any
//	vector of a number              dbimp.Vector[T]
//
// A caller that wants gocql.UUID or a *inf.Dec scans into that type.

// canonical decodes one column into its canonical value. A NULL column gives
// nil.
func canonical(info gocql.TypeInfo, null bool, data []byte) (any, error) {
	if null {
		return nil, nil
	}
	if info.Type() == gocql.TypeBlob {
		// The column is not NULL, so an empty blob is an empty slice and not
		// nil.
		return append([]byte{}, data...), nil
	}
	// gocql decodes into the Go type that it chooses for the type, and
	// normalize turns that value into the canonical one.
	v := reflect.New(reflect.TypeOf(info.Zero()))
	if err := gocql.Unmarshal(info, data, v.Interface()); err != nil {
		return nil, err
	}
	return normalize(info, v.Elem().Interface())
}

// normalize turns v, which gocql decoded for the type info, into the canonical
// value of the type. It does the same for each element of a collection.
func normalize(info gocql.TypeInfo, v any) (any, error) {
	if isNil(v) {
		return nil, nil
	}
	switch info.Type() {
	case gocql.TypeUUID, gocql.TypeTimeUUID:
		if u, ok := v.(gocql.UUID); ok {
			return uuid.UUID(u), nil
		}
	case gocql.TypeTinyInt, gocql.TypeSmallInt, gocql.TypeInt, gocql.TypeBigInt, gocql.TypeCounter:
		return reflect.ValueOf(v).Int(), nil
	case gocql.TypeFloat, gocql.TypeDouble:
		return reflect.ValueOf(v).Float(), nil
	case gocql.TypeDecimal:
		// gocql reads a decimal as an *inf.Dec, whose String is the number.
		if s, ok := v.(fmt.Stringer); ok {
			d, _, err := apd.NewFromString(s.String())
			if err != nil {
				return nil, fmt.Errorf("reading the decimal %s: %w", s, err)
			}
			return d, nil
		}
	case gocql.TypeDate:
		if t, ok := v.(time.Time); ok {
			return dbimp.DateOf(t.UTC()), nil
		}
	case gocql.TypeTime:
		if d, ok := v.(time.Duration); ok {
			return dbimp.LocalTime{
				Hour:       int(d / time.Hour),
				Minute:     int(d % time.Hour / time.Minute),
				Second:     int(d % time.Minute / time.Second),
				Nanosecond: int(d % time.Second),
			}, nil
		}
	case gocql.TypeDuration:
		if d, ok := v.(gocql.Duration); ok {
			return dbimp.Interval{Months: d.Months, Days: d.Days, Nanoseconds: d.Nanoseconds}, nil
		}
	case gocql.TypeInet:
		if ip, ok := v.(net.IP); ok {
			// gocql writes an IPv4 address in 16 bytes. Cassandra tells the two
			// apart by length, and gocql does not pass the length on.
			if v4 := ip.To4(); v4 != nil {
				ip = v4
			}
			addr, ok := netip.AddrFromSlice(ip)
			if !ok {
				return nil, fmt.Errorf("reading an inet of %d bytes: %w", len(ip), dbimp.ErrInvalidValue)
			}
			return addr, nil
		}
	case gocql.TypeList, gocql.TypeSet, gocql.TypeMap:
		if t, ok := info.(gocql.CollectionType); ok {
			if t.Type() == gocql.TypeMap {
				return normalizeMap(t, v)
			}
			return normalizeList(t.Elem, v)
		}
	case gocql.TypeTuple:
		if t, ok := info.(gocql.TupleTypeInfo); ok {
			return normalizeElems(t.Elems, v)
		}
	case gocql.TypeUDT:
		if t, ok := info.(gocql.UDTTypeInfo); ok {
			return normalizeUDT(t, v)
		}
	case gocql.TypeCustom:
		if t, ok := info.(gocql.VectorType); ok {
			return normalizeVector(t, v)
		}
	}
	return v, nil
}

// isNil reports whether v is a nil pointer, slice or map.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return rv.IsNil() && rv.Kind() != reflect.Slice
	}
	return false
}

// normalizeList makes a []any of the canonical values of the elements of a
// slice.
func normalizeList(elem gocql.TypeInfo, v any) (any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return v, nil
	}
	out := make([]any, rv.Len())
	for i := range out {
		e, err := normalize(elem, rv.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		out[i] = e
	}
	return out, nil
}

// normalizeElems makes a []any of the canonical values of the elements of a
// tuple, each with its own type.
func normalizeElems(elems []gocql.TypeInfo, v any) (any, error) {
	list, ok := v.([]any)
	if !ok || len(list) != len(elems) {
		return v, nil
	}
	out := make([]any, len(list))
	for i := range list {
		e, err := normalize(elems[i], list[i])
		if err != nil {
			return nil, err
		}
		out[i] = e
	}
	return out, nil
}

// normalizeUDT makes a map[string]any of the canonical values of the fields of
// a user defined type.
func normalizeUDT(info gocql.UDTTypeInfo, v any) (any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return v, nil
	}
	out := make(map[string]any, len(m))
	for _, f := range info.Elements {
		e, err := normalize(f.Type, m[f.Name])
		if err != nil {
			return nil, err
		}
		out[f.Name] = e
	}
	return out, nil
}

// isTextKey reports whether info is a text type, which is the key of a
// map[string]any.
func isTextKey(info gocql.TypeInfo) bool {
	switch info.Type() {
	case gocql.TypeAscii, gocql.TypeText, gocql.TypeVarchar:
		return true
	}
	return false
}

// normalizeMap makes a map[string]any of the canonical values of a map with a
// text key. A map with any other key stays the map that gocql decoded,
// because no kind of dbimp holds such a map (docs/PLAN.md, open question 15).
func normalizeMap(info gocql.CollectionType, v any) (any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map || !isTextKey(info.Key) {
		return v, nil
	}
	out := make(map[string]any, rv.Len())
	for it := rv.MapRange(); it.Next(); {
		e, err := normalize(info.Elem, it.Value().Interface())
		if err != nil {
			return nil, err
		}
		out[it.Key().String()] = e
	}
	return out, nil
}

// normalizeVector makes a dbimp.Vector of the elements of a vector of numbers,
// and a []any of the canonical values of any other vector.
func normalizeVector(info gocql.VectorType, v any) (any, error) {
	switch s := v.(type) {
	case []float32:
		return dbimp.Vector[float32](s), nil
	case []float64:
		return dbimp.Vector[float64](s), nil
	case []int8:
		return dbimp.Vector[int8](s), nil
	case []int16:
		return dbimp.Vector[int16](s), nil
	case []int32:
		return dbimp.Vector[int32](s), nil
	case []int64:
		return dbimp.Vector[int64](s), nil
	}
	return normalizeList(info.SubType, v)
}

// The canonical Go types.
var (
	stringType   = reflect.TypeFor[string]()
	uuidType     = reflect.TypeFor[uuid.UUID]()
	bytesType    = reflect.TypeFor[[]byte]()
	boolType     = reflect.TypeFor[bool]()
	int64Type    = reflect.TypeFor[int64]()
	float64Type  = reflect.TypeFor[float64]()
	decimalType  = reflect.TypeFor[*apd.Decimal]()
	bigType      = reflect.TypeFor[*big.Int]()
	addrType     = reflect.TypeFor[netip.Addr]()
	timeType     = reflect.TypeFor[time.Time]()
	dateType     = reflect.TypeFor[dbimp.Date]()
	clockType    = reflect.TypeFor[dbimp.LocalTime]()
	intervalType = reflect.TypeFor[dbimp.Interval]()
	listType     = reflect.TypeFor[[]any]()
	mapType      = reflect.TypeFor[map[string]any]()
)

// scanType returns the canonical Go type of a column of type info.
func scanType(info gocql.TypeInfo) reflect.Type {
	switch info.Type() {
	case gocql.TypeAscii, gocql.TypeText, gocql.TypeVarchar:
		return stringType
	case gocql.TypeUUID, gocql.TypeTimeUUID:
		return uuidType
	case gocql.TypeBlob:
		return bytesType
	case gocql.TypeBoolean:
		return boolType
	case gocql.TypeTinyInt, gocql.TypeSmallInt, gocql.TypeInt, gocql.TypeBigInt, gocql.TypeCounter:
		return int64Type
	case gocql.TypeFloat, gocql.TypeDouble:
		return float64Type
	case gocql.TypeDecimal:
		return decimalType
	case gocql.TypeVarint:
		return bigType
	case gocql.TypeInet:
		return addrType
	case gocql.TypeTimestamp:
		return timeType
	case gocql.TypeDate:
		return dateType
	case gocql.TypeTime:
		return clockType
	case gocql.TypeDuration:
		return intervalType
	case gocql.TypeList, gocql.TypeSet, gocql.TypeTuple:
		return listType
	case gocql.TypeMap:
		if t, ok := info.(gocql.CollectionType); ok && isTextKey(t.Key) {
			return mapType
		}
	case gocql.TypeUDT:
		return mapType
	case gocql.TypeCustom:
		if t, ok := info.(gocql.VectorType); ok {
			return vectorType(t)
		}
	}
	return reflect.TypeOf(info.Zero())
}

// vectorType returns the Go type of a vector.
func vectorType(info gocql.VectorType) reflect.Type {
	switch info.SubType.Type() {
	case gocql.TypeFloat:
		return reflect.TypeFor[dbimp.Vector[float32]]()
	case gocql.TypeDouble:
		return reflect.TypeFor[dbimp.Vector[float64]]()
	case gocql.TypeTinyInt:
		return reflect.TypeFor[dbimp.Vector[int8]]()
	case gocql.TypeSmallInt:
		return reflect.TypeFor[dbimp.Vector[int16]]()
	case gocql.TypeInt:
		return reflect.TypeFor[dbimp.Vector[int32]]()
	case gocql.TypeBigInt:
		return reflect.TypeFor[dbimp.Vector[int64]]()
	}
	return listType
}

// bind returns the value that gocql takes for the argument v, and reports
// whether v is one of the canonical types of this driver, which gocql does not
// take as they are. The canonical type of a column, such as dbimp.Date or an
// *apd.Decimal, is also a type that a caller can bind.
func bind(v any) (any, bool) {
	switch v := v.(type) {
	case dbimp.Date:
		return date{v}, true
	case dbimp.LocalTime:
		return time.Duration(v.Hour)*time.Hour + time.Duration(v.Minute)*time.Minute +
			time.Duration(v.Second)*time.Second + time.Duration(v.Nanosecond), true
	case dbimp.Interval:
		return gocql.Duration{Months: v.Months, Days: v.Days, Nanoseconds: v.Nanoseconds}, true
	case netip.Addr:
		return net.IP(v.AsSlice()), true
	case *apd.Decimal:
		if v == nil {
			return nil, true
		}
		return decimal{v}, true
	case apd.Decimal:
		return decimal{&v}, true
	case dbimp.Vector[float32]:
		return []float32(v), true
	case dbimp.Vector[float64]:
		return []float64(v), true
	case dbimp.Vector[int8]:
		return []int8(v), true
	case dbimp.Vector[int16]:
		return []int16(v), true
	case dbimp.Vector[int32]:
		return []int32(v), true
	case dbimp.Vector[int64]:
		return []int64(v), true
	}
	return v, false
}

// date is a dbimp.Date that gocql marshals. gocql writes a time.Time that is the
// zero value, which is the date 0001-01-01, as an empty value, so a date as
// time.Time cannot bind that day. The wire form is the number of days since
// 1970-01-01, plus 2**31, as four bytes.
type date struct {
	d dbimp.Date
}

// MarshalCQL satisfies gocql.Marshaler.
func (d date) MarshalCQL(gocql.TypeInfo) ([]byte, error) {
	days := time.Date(d.d.Year, d.d.Month, d.d.Day, 0, 0, 0, 0, time.UTC).Unix() / secondsPerDay
	if days < math.MinInt32 || days > math.MaxInt32 {
		return nil, fmt.Errorf("writing the date %s: it is outside the range of a CQL date: %w", d.d, dbimp.ErrInvalidValue)
	}
	return binary.BigEndian.AppendUint32(nil, uint32(days+1<<31)), nil
}

// secondsPerDay is the length of a day in a time that has no leap seconds.
const secondsPerDay = 24 * 60 * 60

// decimal is an *apd.Decimal that gocql marshals. gocql marshals a decimal
// column from an *inf.Dec, and the driver does not import that package, so the
// decimal writes its own wire form: the scale as an int32, then the unscaled
// value as a big endian two's complement integer.
type decimal struct {
	d *apd.Decimal
}

// MarshalCQL satisfies gocql.Marshaler.
func (d decimal) MarshalCQL(gocql.TypeInfo) ([]byte, error) {
	if d.d.Form != apd.Finite {
		return nil, fmt.Errorf("writing the decimal %s: it is not a finite number: %w", d.d, dbimp.ErrInvalidValue)
	}
	if d.d.Exponent == math.MinInt32 {
		return nil, fmt.Errorf("writing the decimal %s: its scale does not fit in an int32: %w", d.d, dbimp.ErrInvalidValue)
	}
	unscaled := new(big.Int).Set(d.d.Coeff.MathBigInt())
	if d.d.Negative {
		unscaled.Neg(unscaled)
	}
	scale, err := binary.Append(nil, binary.BigEndian, -d.d.Exponent)
	if err != nil {
		return nil, fmt.Errorf("writing the scale of the decimal %s: %w", d.d, err)
	}
	return append(scale, twosComplement(unscaled)...), nil
}

// twosComplement returns x as the shortest big endian two's complement
// integer, which is what Java BigInteger.toByteArray writes.
func twosComplement(x *big.Int) []byte {
	if x.Sign() >= 0 {
		b := x.Bytes()
		if len(b) == 0 || b[0]&0x80 != 0 {
			b = append([]byte{0}, b...)
		}
		return b
	}
	// For a negative x, add 2**(8n) for the smallest n bytes that hold x. The
	// bits of Not(x), which is -x-1, say how many bytes that is.
	n := (new(big.Int).Not(x).BitLen() + 8) / 8
	mod := new(big.Int).Lsh(big.NewInt(1), uint(8*n))
	b := mod.Add(mod, x).Bytes()
	for len(b) < n {
		b = append([]byte{0}, b...)
	}
	return b
}

// nativeNames holds the name of each native CQL type, in upper case.
var nativeNames = map[gocql.Type]string{
	gocql.TypeAscii:     "ASCII",
	gocql.TypeBigInt:    "BIGINT",
	gocql.TypeBlob:      "BLOB",
	gocql.TypeBoolean:   "BOOLEAN",
	gocql.TypeCounter:   "COUNTER",
	gocql.TypeDecimal:   "DECIMAL",
	gocql.TypeDouble:    "DOUBLE",
	gocql.TypeFloat:     "FLOAT",
	gocql.TypeInt:       "INT",
	gocql.TypeText:      "TEXT",
	gocql.TypeTimestamp: "TIMESTAMP",
	gocql.TypeUUID:      "UUID",
	gocql.TypeVarchar:   "VARCHAR",
	gocql.TypeVarint:    "VARINT",
	gocql.TypeTimeUUID:  "TIMEUUID",
	gocql.TypeInet:      "INET",
	gocql.TypeDate:      "DATE",
	gocql.TypeTime:      "TIME",
	gocql.TypeSmallInt:  "SMALLINT",
	gocql.TypeTinyInt:   "TINYINT",
	gocql.TypeDuration:  "DURATION",
}

// typeName returns the CQL name of info in upper case, such as LIST<INT>. The
// name of a user defined type keeps its case, because a quoted CQL name is
// case sensitive. gocql drops frozen<...> when it reads a type, so the name
// never holds FROZEN.
//
// The driver builds the name itself, because gocql writes a collection as
// map(text, int), which is not CQL.
func typeName(info gocql.TypeInfo) string {
	switch t := info.(type) {
	case gocql.CollectionType:
		switch t.Type() {
		case gocql.TypeList:
			return "LIST<" + typeName(t.Elem) + ">"
		case gocql.TypeSet:
			return "SET<" + typeName(t.Elem) + ">"
		case gocql.TypeMap:
			return "MAP<" + typeName(t.Key) + ", " + typeName(t.Elem) + ">"
		}
	case gocql.TupleTypeInfo:
		names := make([]string, len(t.Elems))
		for i, elem := range t.Elems {
			names[i] = typeName(elem)
		}
		return "TUPLE<" + strings.Join(names, ", ") + ">"
	case gocql.UDTTypeInfo:
		return t.Name
	case gocql.VectorType:
		return "VECTOR<" + typeName(t.SubType) + ", " + strconv.Itoa(t.Dimensions) + ">"
	}
	if name, ok := nativeNames[info.Type()]; ok {
		return name
	}
	return "CUSTOM"
}
