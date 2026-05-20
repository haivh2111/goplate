// Package shared is a compile-harness stub. It exposes the surface that
// goplate-generated code references — enough to compile, no real behaviour.
package shared

// Response is the JSON envelope returned by OK / Created.
type Response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// OK wraps data for a 200 response.
func OK(data any) Response { return Response{Data: data} }

// Created wraps data for a 201 response.
func Created(data any) Response { return Response{Data: data} }

// AppError is the canonical typed error returned by services.
type AppError struct {
	Code int
	Msg  string
}

func (e *AppError) Error() string { return e.Msg }

// ErrNotFound, ErrBadRequest, ErrForbidden, ErrConflict are the helpers
// generated services reach for. Signatures must match what the templates emit.

func ErrNotFound(kind string, id any) error { return &AppError{Code: 404, Msg: kind + " not found"} }
func ErrBadRequest(msg string) error        { return &AppError{Code: 400, Msg: msg} }
func ErrForbidden(msg string) error         { return &AppError{Code: 403, Msg: msg} }
func ErrConflict(msg string) error          { return &AppError{Code: 409, Msg: msg} }
