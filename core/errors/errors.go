// Package errors defines stable machine-readable application errors while
// preserving internal causes for logs and diagnostics.
package errors

import (
	stderrors "errors"
	"fmt"
)

// Error carries a stable code and safe public message.
type Error struct {
	Code, Public string
	Cause        error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Cause)
	}
	return e.Code
}

// Unwrap enables errors.Is and errors.As without exposing causes over HTTP.
func (e *Error) Unwrap() error { return e.Cause }

// New creates a coded error with a safe public message.
func New(code, public string) *Error { return &Error{Code: code, Public: public} }

// Wrap attaches an internal cause to a coded error.
func Wrap(code, public string, cause error) *Error {
	return &Error{Code: code, Public: public, Cause: cause}
}

// As returns a coded error from an error chain.
func As(err error) (*Error, bool) {
	var target *Error
	ok := stderrors.As(err, &target)
	return target, ok
}
