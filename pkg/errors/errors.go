// Package errors defines the small set of domain-error kinds the whole
// backend uses. Handlers at the HTTP edge map Kind -> status code once,
// instead of every layer knowing about HTTP.
package errors

import "fmt"

type Kind int

const (
	KindUnknown Kind = iota
	KindNotFound
	KindConflict
	KindInvalidInput
	KindUnauthorized
	KindForbidden
	KindRateLimited
)

// Error is a domain error carrying enough structure for the HTTP layer to
// pick a status code without string matching.
type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func NotFound(msg string) *Error     { return &Error{Kind: KindNotFound, Message: msg} }
func Conflict(msg string) *Error     { return &Error{Kind: KindConflict, Message: msg} }
func InvalidInput(msg string) *Error { return &Error{Kind: KindInvalidInput, Message: msg} }
func Unauthorized(msg string) *Error { return &Error{Kind: KindUnauthorized, Message: msg} }
func Forbidden(msg string) *Error    { return &Error{Kind: KindForbidden, Message: msg} }
func RateLimited(msg string) *Error  { return &Error{Kind: KindRateLimited, Message: msg} }
func Wrap(kind Kind, msg string, err error) *Error {
	return &Error{Kind: kind, Message: msg, Err: err}
}

func KindOf(err error) Kind {
	var e *Error
	if err == nil {
		return KindUnknown
	}
	if as, ok := err.(*Error); ok {
		e = as
		return e.Kind
	}
	return KindUnknown
}
