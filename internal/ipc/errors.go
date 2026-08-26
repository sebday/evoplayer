package ipc

import "fmt"

const (
	CodeInvalidParams = "invalid_params"
	CodeConflict      = "conflict"
	CodeNotFound      = "not_found"
	CodeUnavailable   = "unavailable"
	CodeUnknownMethod = "unknown_method"
)

type Error struct {
	Code    string
	Message string
	Data    map[string]any
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func AsError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	if e, ok := err.(*Error); ok {
		return e, true
	}
	return nil, false
}

func ErrInvalidParams(format string, args ...any) error {
	return &Error{
		Code:    CodeInvalidParams,
		Message: fmt.Sprintf(format, args...),
	}
}

func ErrConflict(queueRevision uint64) error {
	return &Error{
		Code:    CodeConflict,
		Message: "queue revision conflict",
		Data:    map[string]any{"queue_revision": queueRevision},
	}
}

func ErrNotFound(format string, args ...any) error {
	return &Error{
		Code:    CodeNotFound,
		Message: fmt.Sprintf(format, args...),
	}
}

func ErrUnavailable(format string, args ...any) error {
	return &Error{
		Code:    CodeUnavailable,
		Message: fmt.Sprintf(format, args...),
	}
}

func ErrUnknownMethod(method string) error {
	return &Error{
		Code:    CodeUnknownMethod,
		Message: fmt.Sprintf("unknown method: %s", method),
	}
}
