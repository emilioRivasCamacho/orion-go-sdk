package orequest

import (
	"encoding/json"
)

// Request object
type Request struct {
	TracerData  map[string][]string `json:"tracerData" msgpack:"tracerData"`
	Path        string              `json:"path" msgpack:"path"`
	Params      []byte              `json:"params" msgpack:"params"`
	Meta        map[string]string   `json:"meta" msgpack:"meta"`
	CallTimeout *int                `json:"callTimeout" msgpack:"callTimeout"`
}

// GetParams as type
func (r *Request) GetParams(to interface{}) error {
	return json.Unmarshal(r.Params, to)
}

// GetID as type
func (r *Request) GetID() string {
	return r.Meta["x-trace-id"]
}

// SetParams for type
func (r *Request) SetParams(params interface{}) error {
	b, err := json.Marshal(params)
	r.Params = b
	return err
}
