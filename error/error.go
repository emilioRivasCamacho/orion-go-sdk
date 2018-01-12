package error

import (
	"runtime"

	"github.com/satori/go.uuid"
)

type LineOfCode struct {
	File string
	Line int
}

// Error object
type Error struct {
	ID      string     `json:"id" msgpack:"id"`
	Code    string     `json:"code" msgpack:"code"`
	Message string     `json:"message" msgpack:"message"`
	LOC     LineOfCode `json:"-" msgpack:"-"`
}

// New error object
func New(code string) *Error {
	_, file, line, _ := runtime.Caller(1)
	uid, _ := uuid.NewV4()
	return &Error{
		ID:   uid.String(),
		Code: code,
		LOC: LineOfCode{
			file,
			line,
		},
	}
}

// SetMessage for the error
func (e *Error) SetMessage(msg string) *Error {
	e.Message = msg
	return e
}

// Error returns the error message
func (e Error) Error() string {
	return e.Message
}
