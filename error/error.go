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
	LOC     LineOfCode `json:"LOC" msgpack:"-"`
}

// New error object
func New(code string) *Error {

	uid, _ := uuid.NewV4()
	return &Error{
		ID:   uid.String(),
		Code: code,
		LOC:  GenerateLOC(1),
	}
}

// SetMessage for the error
func (e *Error) SetMessage(msg string) *Error {
	e.Message = msg
	return e
}

func (e *Error) SetLineOfCode(loc LineOfCode) *Error {
	e.LOC = loc
	return e
}

// Error returns the error message
func (e Error) Error() string {
	return e.Message
}

func GenerateLOC(depth int) LineOfCode {
	_, file, line, _ := runtime.Caller(depth + 1)
	return LineOfCode{
		file,
		line,
	}
}
