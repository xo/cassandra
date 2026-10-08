package cassandra //nolint:testpackage // The tables read the types of the driver, which are not exported.

import (
	"testing"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp/dbimptest"
)

// doc is the product document of the driver.
const doc = "docs/CASSANDRA.md"

// TestTables compares the type table and the interface table of step 10 of
// the driver guide of dbimp in doc with the code. Run it with DBIMP_UPDATE=1 to
// write them.
func TestTables(t *testing.T) {
	t.Parallel()
	row := func(wire, goType string, info gocql.TypeInfo) dbimptest.TypeRow {
		r := &rows{cols: []gocql.ColumnInfo{column("c", info)}}
		return dbimptest.TypeRow{
			Wire:         wire,
			Go:           goType,
			ScanType:     r.ColumnTypeScanType(0).String(),
			DatabaseType: r.ColumnTypeDatabaseTypeName(0),
			// Every column that is not a primary key can be NULL, and gocql
			// does not say which a column is.
			Nullable: true,
		}
	}
	dbimptest.TypeTable(t, doc, dbimptest.Kinds(t, []dbimptest.TypeRow{
		row("ascii", "string", native(gocql.TypeAscii)),
		row("text", "string", native(gocql.TypeText)),
		row("varchar", "string", native(gocql.TypeVarchar)),
		row("boolean", "bool", native(gocql.TypeBoolean)),
		row("tinyint", "int64", native(gocql.TypeTinyInt)),
		row("smallint", "int64", native(gocql.TypeSmallInt)),
		row("int", "int64", native(gocql.TypeInt)),
		row("bigint", "int64", native(gocql.TypeBigInt)),
		row("counter", "int64", native(gocql.TypeCounter)),
		row("float", "float64", native(gocql.TypeFloat)),
		row("double", "float64", native(gocql.TypeDouble)),
		row("decimal", "*apd.Decimal", native(gocql.TypeDecimal)),
		row("varint", "*big.Int", native(gocql.TypeVarint)),
		row("blob", "[]byte", native(gocql.TypeBlob)),
		row("uuid", "uuid.UUID", native(gocql.TypeUUID)),
		row("timeuuid", "uuid.UUID", native(gocql.TypeTimeUUID)),
		row("inet", "netip.Addr", native(gocql.TypeInet)),
		row("timestamp", "time.Time", native(gocql.TypeTimestamp)),
		row("date", "dbimp.Date", native(gocql.TypeDate)),
		row("time", "dbimp.LocalTime", native(gocql.TypeTime)),
		row("duration", "dbimp.Interval", native(gocql.TypeDuration)),
		row("list", "[]any", parse("list<int>")),
		row("set", "[]any", parse("set<text>")),
		row("map with a text key", "map[string]any", parse("map<text, int>")),
		row("map with another key", "the map that gocql chooses", parse("map<int, text>")),
		row("tuple", "[]any", gocql.TupleTypeInfo{Elems: []gocql.TypeInfo{native(gocql.TypeInt), native(gocql.TypeText)}}),
		// The table writes the type name in upper case. The driver keeps the case
		// of the name of a type, because a quoted CQL name is case sensitive.
		row("user defined type", "map[string]any", gocql.UDTTypeInfo{Keyspace: "ks", Name: "ADDRESS", Elements: udt().Elements}),
		row("vector of float", "dbimp.Vector[float32]", parse("vector<float, 3>")),
		row("vector of double", "dbimp.Vector[float64]", parse("vector<double, 3>")),
		row("vector of int", "dbimp.Vector[int32]", parse("vector<int, 3>")),
		row("vector of bigint", "dbimp.Vector[int64]", parse("vector<bigint, 3>")),
		row("vector of other", "[]any", parse("vector<text, 3>")),
	}, typeKinds))
	dbimptest.InterfaceTable(t, doc, map[string]string{
		"driver.DriverContext":                  "OpenConnector parses the DSN once, for every connection.",
		"driver.Connector":                      "The connector owns the one gocql session, which every connection shares (D17).",
		"io.Closer on the connector":            "Close closes the session.",
		"driver.Pinger":                         "Ping reads the release version of the node that answers.",
		"driver.SessionResetter":                "It returns driver.ErrBadConn after the connector is closed, so that database/sql discards the connection. A connection holds no other state.",
		"driver.Validator":                      "It reports whether the connector is still open.",
		"driver.NamedValueChecker":              "It keeps an Option, and the values that the driver binds with a type of its own: a decimal, the types of dbimp, an address and a UUID (D25).",
		"driver.QueryerContext":                 "gocql binds each argument at its ? marker.",
		"driver.ExecerContext":                  "RowsAffected and LastInsertId fail, because Cassandra reports neither.",
		"driver.ConnPrepareContext":             "A prepared statement holds the text, and gocql prepares and caches it when it runs.",
		"driver.ConnBeginTx":                    "BeginTx fails with dbimp.ErrNotSupported, because CQL has no transactions (D9).",
		"driver.RowsColumnScanner":              "gocql decodes straight into the destination of the caller, and a canonical value reaches any, a Scanner and the types of dbimp.",
		"driver.RowsNextResultSet":              "A statement has one result.",
		"driver.RowsColumnTypeScanType":         "From the type of the column, with one Go type for each type.",
		"driver.RowsColumnTypeDatabaseTypeName": "The CQL type in upper case, such as LIST<INT>.",
		"driver.RowsColumnTypeLength":           "No CQL type has a length.",
		"driver.RowsColumnTypeNullable":         "It reports that the nullability is not known, because gocql does not say which column is a primary key.",
		"driver.RowsColumnTypePrecisionScale":   "A decimal column has no precision or scale in its type.",
	}, newTestConnector(&fakeSession{}), Driver{}, &conn{}, &rows{}, &stmt{})
}

// typeKinds are the kinds of the types document of dbimp of the types of the
// driver.
var typeKinds = map[string]string{
	"ascii":                "string",
	"text":                 "string",
	"varchar":              "string",
	"boolean":              "boolean",
	"tinyint":              "integer",
	"smallint":             "integer",
	"int":                  "integer",
	"bigint":               "integer",
	"counter":              "integer",
	"float":                "float",
	"double":               "float",
	"decimal":              "decimal",
	"varint":               "big integer",
	"blob":                 "binary",
	"uuid":                 "uuid",
	"timeuuid":             "uuid",
	"inet":                 "ip address",
	"timestamp":            "timestamp",
	"date":                 "date",
	"time":                 "time of day",
	"duration":             "interval",
	"list":                 "array",
	"set":                  "set",
	"map with a text key":  "map",
	"map with another key": "map",
	"tuple":                "tuple",
	"user defined type":    "map",
	"vector of float":      "vector",
	"vector of double":     "vector",
	"vector of int":        "vector",
	"vector of bigint":     "vector",
	"vector of other":      "array",
}
