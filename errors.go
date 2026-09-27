package cql

// Error is an error that the driver reports.
type Error string

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}

// Error values.
const (
	// ErrNoTransactions is returned by BeginTx. CQL has no transactions.
	ErrNoTransactions Error = "transactions are not supported"
	// ErrNamedArgs is returned for an argument from sql.Named. gocql has no
	// call that binds a value by name.
	ErrNamedArgs Error = "named arguments are not supported"
	// ErrUnsupportedArg is returned for an argument that gocql can never bind,
	// such as a channel or a function.
	ErrUnsupportedArg Error = "argument type is not supported"
	// ErrConnectorClosed is returned by Connect after Close.
	ErrConnectorClosed Error = "connector is closed"
	// ErrInvalidDSN is returned by ParseDSN and FormatDSN, wrapped with the
	// part of the DSN that is not valid.
	ErrInvalidDSN Error = "invalid dsn"
	// ErrNoContext is returned by the methods of the driver.Stmt interface
	// that take no context. database/sql never calls them, because the driver
	// has the forms that take a context.
	ErrNoContext Error = "a call with no context is not supported"
)
