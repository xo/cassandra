package cassandra

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"math/big"
	"net/netip"
	"reflect"
	"uuid"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp"
)

// capture holds one value of the current row, as gocql read it. gocql calls
// UnmarshalCQL on any scan destination that implements gocql.Unmarshaler, so
// the driver gets the type and the bytes of each column without decoding
// them.
type capture struct {
	info gocql.TypeInfo
	// null is true when the value is NULL. It is kept apart from data,
	// because an empty value is not NULL.
	null bool
	// data holds the value in the CQL wire format. The buffer is reused from
	// row to row.
	data []byte
}

// UnmarshalCQL satisfies gocql.Unmarshaler.
func (c *capture) UnmarshalCQL(info gocql.TypeInfo, data []byte) error {
	c.info, c.null = info, data == nil
	// gocql passes a slice of its frame buffer, which it reuses.
	c.data = append(c.data[:0], data...)
	return nil
}

// bytes returns data, or nil for a NULL value.
func (c *capture) bytes() []byte {
	if c.null {
		return nil
	}
	return c.data
}

// rows is the result of a query. It implements driver.RowsColumnScanner, so
// database/sql passes the scan destination of the caller to ScanColumn, and
// gocql can decode straight into a *[]string, a *map[string]int or a
// *gocql.UUID.
type rows struct {
	it iterator
	// cancel releases the timeout of the statement. It is never nil.
	cancel context.CancelFunc
	cols   []gocql.ColumnInfo
	names  []string

	// caps holds the captures of each column. A tuple column has one capture
	// for each element, because gocql scans each element into its own
	// destination. Every other column has one.
	caps [][]capture
	// dest holds a pointer to every capture, in the order that gocql scans
	// them.
	dest []any
	// read counts the rows that NextRow returned.
	read int
}

func newRows(it iterator) *rows {
	cols := it.columns()
	r := &rows{
		it:     it,
		cancel: func() {},
		cols:   cols,
		names:  make([]string, len(cols)),
		caps:   make([][]capture, len(cols)),
	}
	for i, col := range cols {
		r.names[i] = col.Name
		n := 1
		if t, ok := col.TypeInfo.(gocql.TupleTypeInfo); ok {
			n = len(t.Elems)
		}
		r.caps[i] = make([]capture, n)
		for j := range r.caps[i] {
			r.dest = append(r.dest, &r.caps[i][j])
		}
	}
	return r
}

// Columns returns the names of the columns.
func (r *rows) Columns() []string {
	return r.names
}

// Close releases the rows. It returns the error from gocql, if the rows ended
// with one that NextRow did not return.
func (r *rows) Close() error {
	if r.it == nil {
		return nil
	}
	err := r.it.close()
	r.it = nil
	r.cancel()
	if err != nil {
		return fmt.Errorf("closing rows: %w", err)
	}
	return nil
}

// NextRow reads the next row. It returns io.EOF at the end of the rows, and
// the error from gocql when the rows end with one. gocql reads the next page
// here, with the context of the query.
func (r *rows) NextRow() error {
	if r.it == nil {
		return io.EOF
	}
	if r.it.scan(r.dest...) {
		r.read++
		return nil
	}
	err := r.it.close()
	r.it = nil
	r.cancel()
	switch {
	case err != nil && r.read > 0:
		// At least one row reached the caller, so the result is incomplete
		// (dbimp D21).
		return fmt.Errorf("reading rows after %d rows: %w: %w", r.read, dbimp.ErrIncomplete, err)
	case err != nil:
		return fmt.Errorf("reading rows: %w", err)
	}
	return io.EOF
}

// Next reads the next row into dest as canonical values. database/sql calls
// NextRow and ScanColumn instead. Next is for code that uses the driver
// without database/sql.
func (r *rows) Next(dest []driver.Value) error {
	if err := r.NextRow(); err != nil {
		return err
	}
	for i := range dest {
		v, err := r.value(i)
		if err != nil {
			return fmt.Errorf("decoding column %q: %w", r.names[i], err)
		}
		dest[i] = v
	}
	return nil
}

// ScanColumn decodes column index of the current row into dest.
//
// A sql.Scanner, *any, *sql.RawBytes, a *uuid.UUID, a **uuid.UUID, a pointer
// to a type of dbimp or a netip.Addr, and a tuple column take the canonical
// value, which dbimp.Assign converts. gocql
// cannot decode into the standard uuid.UUID, and the canonical value of a
// uuid column is one (D25). Any other destination goes to
// gocql. When gocql refuses a pointer to a basic kind, such as a *float64 for
// an int column, sql.ConvertAssign tries the canonical value. gocql writes
// nothing to a scalar that it refuses, so the second try is safe. gocql can
// write part of a slice, a map or a struct before it fails, so the driver does
// not try again for those.
func (r *rows) ScanColumn(sc driver.ScanContext, index int, dest any) error {
	if r.cols[index].TypeInfo.Type() == gocql.TypeTuple {
		return r.convert(sc, index, dest)
	}
	c := &r.caps[index][0]
	switch dest.(type) {
	case nil, sql.Scanner, *any, *sql.RawBytes, *uuid.UUID, **uuid.UUID,
		*dbimp.Date, *dbimp.LocalTime, *dbimp.Interval, *netip.Addr:
		return r.convert(sc, index, dest)
	case gocql.Unmarshaler:
		// A type of the caller that decodes itself also decides what NULL is.
		return gocql.Unmarshal(c.info, c.bytes(), dest)
	}
	kind := elemKind(dest)
	if c.null {
		switch kind {
		case reflect.Pointer, reflect.Slice, reflect.Map:
			// gocql sets the pointer, slice or map to nil. Cassandra stores
			// an empty collection as NULL, so this is not an error.
			return gocql.Unmarshal(c.info, nil, dest)
		}
		// This returns the same error for NULL that database/sql returns
		// with every other driver, such as converting NULL to string is
		// unsupported.
		return sql.ConvertAssign(sc, dest, nil)
	}
	err := gocql.Unmarshal(c.info, c.data, dest)
	if err != nil && isBasic(kind) {
		if v, cerr := canonical(c.info, false, c.data); cerr == nil {
			if assign(sc, dest, v) == nil {
				return nil
			}
		}
	}
	return err
}

// ColumnTypeDatabaseTypeName returns the CQL type of column index in upper
// case, such as TEXT or LIST<INT>.
func (r *rows) ColumnTypeDatabaseTypeName(index int) string {
	return typeName(r.cols[index].TypeInfo)
}

// ColumnTypeScanType returns the Go type that *any receives for column index.
func (r *rows) ColumnTypeScanType(index int) reflect.Type {
	return scanType(r.cols[index].TypeInfo)
}

// ColumnTypeNullable reports that the nullability of the column is unknown. A
// regular column can be NULL and a primary key column cannot, but gocql does
// not say which a column is.
func (r *rows) ColumnTypeNullable(int) (bool, bool) {
	return false, false
}

// convert decodes column index into its canonical value and passes it to
// sql.ConvertAssign.
func (r *rows) convert(sc driver.ScanContext, index int, dest any) error {
	v, err := r.value(index)
	if err != nil {
		return err
	}
	return assign(sc, dest, v)
}

// assign stores the canonical value src in dest. A *big.Int and a netip.Addr
// go to a destination of their own type, or to a *any, and to any other
// destination as their text. dbimp.Assign stores every other value.
func assign(sc driver.ScanContext, dest, src any) error {
	switch s := src.(type) {
	case *big.Int:
		switch d := dest.(type) {
		case *big.Int:
			d.Set(s)
			return nil
		case *any:
			*d = s
			return nil
		}
		src = s.String()
	case netip.Addr:
		switch d := dest.(type) {
		case *netip.Addr:
			*d = s
			return nil
		case *any:
			*d = s
			return nil
		}
		src = s.String()
	}
	return dbimp.Assign(sc, dest, src)
}

// value returns the canonical value of column index. A tuple is an []any of
// the canonical values of its elements. gocql reports a NULL tuple as a tuple
// whose elements are all NULL, so the two cannot be told apart.
func (r *rows) value(index int) (any, error) {
	caps := r.caps[index]
	if r.cols[index].TypeInfo.Type() != gocql.TypeTuple {
		return canonical(caps[0].info, caps[0].null, caps[0].data)
	}
	v := make([]any, len(caps))
	for i := range caps {
		var err error
		if v[i], err = canonical(caps[i].info, caps[i].null, caps[i].data); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// elemKind returns the kind that dest points to, or reflect.Invalid when dest
// is not a pointer.
func elemKind(dest any) reflect.Kind {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return reflect.Invalid
	}
	return v.Elem().Kind()
}

// isBasic reports whether k is a bool, an integer, a float or a string.
func isBasic(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
