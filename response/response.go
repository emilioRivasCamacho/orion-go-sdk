package response

import (
	"github.com/betit/orion-go-sdk/codec/msgpack"
	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/betit/orion-go-sdk/interfaces"
)

// Response from the service
type Response struct {
	Payload []byte        `json:"payload" msgpack:"payload"`
	Error   *oerror.Error `json:"error" msgpack:"error"`
}

var codec = msgpack.New()

// ParsePayload as type
func (r *Response) ParsePayload(to interface{}) error {
	return codec.Decode(r.Payload, to)
}

// SetPayload for type
func (r *Response) SetPayload(payload interface{}) error {
	b, err := codec.Encode(payload)
	r.Payload = b
	return err
}

// GetError for res
func (r Response) GetError() *oerror.Error {
	return r.Error
}

// SetError for response
func (r *Response) SetError(e *oerror.Error) interfaces.Response {
	r.Error = e
	return r
}
