package request

import (
	"github.com/betit/orion-go-sdk/codec/msgpack"
)

// Request object
type Request struct {
	TracerData  map[string][]string `json:"tracerData" msgpack:"tracerData"`
	Path        string              `json:"path" msgpack:"path"`
	Params      []byte              `json:"params" msgpack:"params"`
	Meta        map[string]string   `json:"meta" msgpack:"meta"`
	CallTimeout *int                `json:"callTimeout" msgpack:"callTimeout"`
}

var codec = msgpack.New()

// GetParams as type
func (r *Request) GetParams(to interface{}) error {
	return codec.Decode(r.Params, to)
}

// GetID as type
func (r *Request) GetID() string {
	return r.Meta["x-trace-id"]
}

// SetParams for type
func (r *Request) SetParams(params interface{}) error {
	b, err := codec.Encode(params)
	r.Params = b
	return err
}
