package response

import (
	"encoding/json"

	oerror "github.com/betit/orion-go-sdk/error"
)

// Response from the service
type Response struct {
	Payload []byte        `json:"payload" msgpack:"payload"`
	Error   *oerror.Error `json:"error" msgpack:"error"`
}

// GetPayload as type
func (r *Response) GetPayload(to interface{}) error {
	return json.Unmarshal(r.Payload, to)
}

// SetPayload for type
func (r *Response) SetPayload(payload interface{}) error {
	b, err := json.Marshal(payload)
	r.Payload = b
	return err
}
