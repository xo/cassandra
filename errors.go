package cassandra

// Error is an error that the driver reports.
type Error string

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}

// Error values. The other errors of the driver wrap the errors of dbimp: a DSN
// wraps dbimp.ErrScheme, dbimp.ErrUnknownKey, dbimp.ErrRepeatedKey or
// dbimp.ErrInvalidValue, and a feature that Cassandra has no form of, such as a
// transaction or a named argument, wraps dbimp.ErrNotSupported.
const (
	// ErrConnectorClosed is returned by Connect after Close.
	ErrConnectorClosed Error = "connector is closed"
)
