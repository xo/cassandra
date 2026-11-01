package cassandra

import (
	"errors"
	"strings"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

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
	// ErrAuthentication is a server that refused the credential. An error
	// that the driver returns for a wrong username or password matches it
	// with errors.Is, and has the method AuthenticationRefused.
	ErrAuthentication Error = "authentication refused"
)

// AuthenticationRefused reports whether err is ErrAuthentication. usql asks
// an error for it, to decide that the server refused the credential. usql
// declares the interface, so this package imports nothing of it.
func (err Error) AuthenticationRefused() bool {
	return err == ErrAuthentication
}

// authError is an error from gocql that says the server refused the
// credential. It keeps the message of gocql.
type authError struct {
	err error
}

// Error satisfies the error interface.
func (e *authError) Error() string { return e.err.Error() }

// Unwrap returns the error from gocql.
func (e *authError) Unwrap() error { return e.err }

// Is reports whether target is ErrAuthentication.
func (e *authError) Is(target error) bool { return target == ErrAuthentication }

// AuthenticationRefused returns true.
func (e *authError) AuthenticationRefused() bool { return true }

// refusal returns err wrapped as an authError when the server refused the
// credential, and err as it is otherwise. gocql gives the typed error
// *gocql.RequestErrCredentials once a session exists. While it creates the
// session, gocql formats the error with %v and the type is lost, so the text
// of the server is the only sign left.
func refusal(err error) error {
	if err == nil {
		return nil
	}
	var c *gocql.RequestErrCredentials
	if errors.As(err, &c) || strings.Contains(err.Error(), "password are incorrect") {
		return &authError{err: err}
	}
	return err
}
