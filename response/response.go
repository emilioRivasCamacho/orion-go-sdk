package response

import (
	"github.com/betit/orion-go-sdk/codec/msgpack"
	oerror "github.com/betit/orion-go-sdk/error"
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
