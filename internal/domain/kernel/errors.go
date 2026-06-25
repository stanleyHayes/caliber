package kernel

import (
	"errors"
	"fmt"
)

// Kind classifies a domain error so adapters can map it (e.g. to gRPC/HTTP codes).
type Kind int

// Error kinds, mapped by adapters to transport-level status codes.
const (
	KindInternal Kind = iota
	KindInvalid
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
)

// Error is the typed domain error carrying a Kind, a message, and an optional cause.
type Error struct {
	Kind Kind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

// Unwrap exposes the wrapped cause for errors.Is/As.
func (e *Error) Unwrap() error { return e.Err }

func newErr(k Kind, msg string) *Error { return &Error{Kind: k, Msg: msg} }

// Invalid builds a KindInvalid error (failed validation / bad input).
func Invalid(msg string) *Error { return newErr(KindInvalid, msg) }

// NotFound builds a KindNotFound error.
func NotFound(msg string) *Error { return newErr(KindNotFound, msg) }

// Conflict builds a KindConflict error (e.g. uniqueness violation).
func Conflict(msg string) *Error { return newErr(KindConflict, msg) }

// Unauthorized builds a KindUnauthorized error (not authenticated).
func Unauthorized(msg string) *Error { return newErr(KindUnauthorized, msg) }

// Forbidden builds a KindForbidden error (authenticated but not permitted).
func Forbidden(msg string) *Error { return newErr(KindForbidden, msg) }

// Invalidf builds a KindInvalid error from a format string.
func Invalidf(format string, a ...any) *Error { return newErr(KindInvalid, fmt.Sprintf(format, a...)) }

// Wrap annotates an existing error with a kind and message.
func Wrap(err error, k Kind, msg string) *Error { return &Error{Kind: k, Msg: msg, Err: err} }

// KindOf returns the Kind of err, or KindInternal if err is not a *Error.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}
