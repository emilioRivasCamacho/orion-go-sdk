package error

import "github.com/satori/go.uuid"

// Error object
type Error struct {
	ID      string `json:"id" msgpack:"id"`
	Code    string `json:"code" msgpack:"code"`
	Message string `json:"message" msgpack:"message"`
}

// New error object
func New(code string) *Error {
	return &Error{
		ID:   uuid.NewV4().String(),
		Code: code,
	}
}

// SetMessage for the error
func (e *Error) SetMessage(msg string) *Error {
	e.Message = msg
	return e
}
