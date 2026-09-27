package cql

import (
	"math/big"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"gopkg.in/inf.v0"
)

// The canonical value of a column is what *any receives, what a sql.Scanner
// receives, what Next returns, and what ColumnTypeScanType reports. One CQL
// type has one canonical Go type in all four places. The basic types follow
// the driver.Value set, so that every sql.Scanner in the standard library
// accepts them. An inet, a varint and a decimal become strings for the same
// reason. A uuid and a timeuuid become the standard uuid.UUID (D25). A caller
// that wants gocql.UUID or *inf.Dec scans into that type.

// canonical decodes one column into its canonical value. A NULL column gives
// nil.
func canonical(info gocql.TypeInfo, null bool, data []byte) (any, error) {
	if null {
		return nil, nil
	}
	switch info.Type() {
	case gocql.TypeAscii, gocql.TypeText, gocql.TypeVarchar:
		var v string
		return v, gocql.Unmarshal(info, data, &v)
	case gocql.TypeUUID, gocql.TypeTimeUUID:
		var v gocql.UUID
		if err := gocql.Unmarshal(info, data, &v); err != nil {
			return nil, err
		}
		return uuid.UUID(v), nil
	case gocql.TypeBlob:
		// The column is not NULL, so an empty blob is an empty slice and not
		// nil.
		return append([]byte{}, data...), nil
	case gocql.TypeBoolean:
		var v bool
		return v, gocql.Unmarshal(info, data, &v)
	case gocql.TypeTinyInt, gocql.TypeSmallInt, gocql.TypeInt, gocql.TypeBigInt, gocql.TypeCounter:
		var v int64
		return v, gocql.Unmarshal(info, data, &v)
	case gocql.TypeFloat:
		var v float32
		if err := gocql.Unmarshal(info, data, &v); err != nil {
			return nil, err
		}
		return float64(v), nil
	case gocql.TypeDouble:
		var v float64
		return v, gocql.Unmarshal(info, data, &v)
	case gocql.TypeVarint:
		var v *big.Int
		if err := gocql.Unmarshal(info, data, &v); err != nil || v == nil {
			return nil, err
		}
		return v.String(), nil
	case gocql.TypeDecimal:
		var v *inf.Dec
		if err := gocql.Unmarshal(info, data, &v); err != nil || v == nil {
			return nil, err
		}
		return v.String(), nil
	case gocql.TypeTimestamp, gocql.TypeDate:
		var v time.Time
		return v, gocql.Unmarshal(info, data, &v)
	case gocql.TypeTime:
		var v time.Duration
		return v, gocql.Unmarshal(info, data, &v)
	case gocql.TypeDuration:
		var v gocql.Duration
		return v, gocql.Unmarshal(info, data, &v)
	case gocql.TypeInet:
		var v net.IP
		if err := gocql.Unmarshal(info, data, &v); err != nil || v == nil {
			return nil, err
		}
		return v.String(), nil
	}
	// A collection, a user defined type or a vector decodes into the Go type
	// that gocql chooses for it, such as []int or map[string]int.
	v := reflect.New(reflect.TypeOf(info.Zero()))
	if err := gocql.Unmarshal(info, data, v.Interface()); err != nil {
		return nil, err
	}
	return v.Elem().Interface(), nil
}

// The canonical Go types that are not the zero value of the gocql type.
var (
	stringType   = reflect.TypeFor[string]()
	uuidType     = reflect.TypeFor[uuid.UUID]()
	bytesType    = reflect.TypeFor[[]byte]()
	boolType     = reflect.TypeFor[bool]()
	int64Type    = reflect.TypeFor[int64]()
	float64Type  = reflect.TypeFor[float64]()
	timeType     = reflect.TypeFor[time.Time]()
	durationType = reflect.TypeFor[time.Duration]()
	cqlDuration  = reflect.TypeFor[gocql.Duration]()
	tupleType    = reflect.TypeFor[[]any]()
)

// scanType returns the canonical Go type of a column of type info.
func scanType(info gocql.TypeInfo) reflect.Type {
	switch info.Type() {
	case gocql.TypeAscii, gocql.TypeText, gocql.TypeVarchar,
		gocql.TypeVarint, gocql.TypeDecimal, gocql.TypeInet:
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
	case gocql.TypeTimestamp, gocql.TypeDate:
		return timeType
	case gocql.TypeTime:
		return durationType
	case gocql.TypeDuration:
		return cqlDuration
	case gocql.TypeTuple:
		return tupleType
	}
	return reflect.TypeOf(info.Zero())
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
