package response

import (
	"github.com/gig/orion-go-sdk/codec/msgpack"
	oerror "github.com/gig/orion-go-sdk/error"
	"github.com/gig/orion-go-sdk/interfaces"
)

// Response from the service
type Response struct {
	// Empty json tags because we need to omit those fields when generating the docs
	// and we do not plan to support json
	Payload []byte        `json:"-" msgpack:"payload"`
	Error   *oerror.Error `json:"-" msgpack:"error"`
}

var codec = msgpack.New()

// New reponse
func New() *Response {
	return &Response{}
}

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
