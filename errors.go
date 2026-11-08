package cassandra

import (
	"errors"
	"fmt"
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

// serverError is an error from gocql that the driver classifies. It keeps the
// message of gocql. refused is true when the server refused the credential.
type serverError struct {
	err     error
	refused bool
}

// Error satisfies the error interface.
func (e *serverError) Error() string { return e.err.Error() }

// Unwrap returns the error from gocql.
func (e *serverError) Unwrap() error { return e.err }

// Is reports whether target is ErrAuthentication and the server refused the
// credential.
func (e *serverError) Is(target error) bool { return e.refused && target == ErrAuthentication }

// AuthenticationRefused reports whether the server refused the credential.
func (e *serverError) AuthenticationRefused() bool { return e.refused }

// ErrVerbose returns every field that the server gave, in the form of cqlsh,
// such as
//
//	Error from server: code=2200 [Invalid query] message="table x does not exist"
//
// usql asks an error for it, for \errverbose, and declares the interface
// itself, so this package imports nothing of usql. A server gives no position
// in the statement, so query is not used. It returns "" for an error that has
// no reply of the server, such as one from before the session exists.
func (e *serverError) ErrVerbose(string) string {
	r, ok := errors.AsType[gocql.RequestError](e.err)
	if !ok {
		return ""
	}
	s := fmt.Sprintf("Error from server: code=%04x [%s] message=%q", r.Code(), codeName(r.Code()), r.Message())
	if info := fields(r); len(info) != 0 {
		s += " info={" + strings.Join(info, ", ") + "}"
	}
	return s
}

// codeName returns the name that cqlsh gives to an error code of the protocol.
func codeName(code int) string {
	switch code {
	case gocql.ErrCodeServer:
		return "Server error"
	case gocql.ErrCodeProtocol:
		return "Protocol error"
	case gocql.ErrCodeCredentials:
		return "Bad credentials"
	case gocql.ErrCodeUnavailable:
		return "Unavailable exception"
	case gocql.ErrCodeOverloaded:
		return "Overloaded"
	case gocql.ErrCodeBootstrapping:
		return "Is_bootstrapping"
	case gocql.ErrCodeTruncate:
		return "Truncate_error"
	case gocql.ErrCodeWriteTimeout:
		return "Write_timeout"
	case gocql.ErrCodeReadTimeout:
		return "Read_timeout"
	case gocql.ErrCodeReadFailure:
		return "Read_failure"
	case gocql.ErrCodeFunctionFailure:
		return "Function_failure"
	case gocql.ErrCodeWriteFailure:
		return "Write_failure"
	case gocql.ErrCodeCDCWriteFailure:
		return "CDC_write_failure"
	case gocql.ErrCodeCASWriteUnknown:
		return "CAS_write_unknown"
	case gocql.ErrCodeSyntax:
		return "Syntax error"
	case gocql.ErrCodeUnauthorized:
		return "Unauthorized"
	case gocql.ErrCodeInvalid:
		return "Invalid query"
	case gocql.ErrCodeConfig:
		return "Config error"
	case gocql.ErrCodeAlreadyExists:
		return "Already exists"
	case gocql.ErrCodeUnprepared:
		return "Unprepared"
	}
	return "Unknown"
}

// fields returns the fields of a reply of the server that the reply holds
// besides its code and message, as the names of cqlsh write them.
func fields(r gocql.RequestError) []string {
	switch e := r.(type) {
	case *gocql.RequestErrUnavailable:
		return []string{
			fmt.Sprintf("consistency: %s", e.Consistency),
			fmt.Sprintf("required_replicas: %d", e.Required),
			fmt.Sprintf("alive_replicas: %d", e.Alive),
		}
	case *gocql.RequestErrWriteTimeout:
		return []string{
			fmt.Sprintf("consistency: %s", e.Consistency),
			fmt.Sprintf("required_responses: %d", e.BlockFor),
			fmt.Sprintf("received_responses: %d", e.Received),
			"write_type: " + e.WriteType,
		}
	case *gocql.RequestErrReadTimeout:
		return []string{
			fmt.Sprintf("consistency: %s", e.Consistency),
			fmt.Sprintf("required_responses: %d", e.BlockFor),
			fmt.Sprintf("received_responses: %d", e.Received),
			fmt.Sprintf("data_retrieved: %t", e.DataPresent != 0),
		}
	case *gocql.RequestErrWriteFailure:
		return []string{
			fmt.Sprintf("consistency: %s", e.Consistency),
			fmt.Sprintf("required_responses: %d", e.BlockFor),
			fmt.Sprintf("received_responses: %d", e.Received),
			fmt.Sprintf("failures: %d", e.NumFailures),
			"write_type: " + e.WriteType,
		}
	case *gocql.RequestErrReadFailure:
		return []string{
			fmt.Sprintf("consistency: %s", e.Consistency),
			fmt.Sprintf("required_responses: %d", e.BlockFor),
			fmt.Sprintf("received_responses: %d", e.Received),
			fmt.Sprintf("failures: %d", e.NumFailures),
			fmt.Sprintf("data_retrieved: %t", e.DataPresent),
		}
	case *gocql.RequestErrAlreadyExists:
		return []string{
			"keyspace: " + e.Keyspace,
			"table: " + e.Table,
		}
	case *gocql.RequestErrFunctionFailure:
		return []string{
			"keyspace: " + e.Keyspace,
			"function: " + e.Function,
			"arg_types: " + strings.Join(e.ArgTypes, ", "),
		}
	}
	return nil
}

// wrap returns err as a serverError when it is a reply of the server, or says
// that the server refused the credential, and err as it is otherwise. gocql
// gives the typed error *gocql.RequestErrCredentials once a session exists.
// While it creates the session, gocql formats the error with %v and the type
// is lost, so the text of the error is the only sign left of a refusal.
func wrap(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[*gocql.RequestErrCredentials](err); ok {
		return &serverError{err: err, refused: true}
	}
	msg := err.Error()
	for _, text := range refusalTexts {
		if strings.Contains(msg, text) {
			return &serverError{err: err, refused: true}
		}
	}
	if _, ok := errors.AsType[gocql.RequestError](err); ok {
		return &serverError{err: err}
	}
	return err
}

// refusalTexts are the messages that mark a refused credential once gocql has
// lost the type of the error: a wrong password, a password that the client did
// not send, and a server that wants authentication where the client has none.
var refusalTexts = []string{
	"password are incorrect",
	"Password must not be null",
	"authentication required (using",
}
