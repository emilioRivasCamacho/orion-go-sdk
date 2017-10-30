package request

import (
	"github.com/betit/orion-go-sdk/codec/msgpack"
	"github.com/betit/orion-go-sdk/interfaces"
)

// Request object
type Request struct {
	TracerData map[string][]string `json:"tracerData" msgpack:"tracerData"`
	Path       string              `json:"path" msgpack:"path"`
	Params     []byte              `json:"params" msgpack:"params"`
	Meta       map[string]string   `json:"meta" msgpack:"meta"`
	Timeout    *int                `json:"timeout" msgpack:"timeout"`
}

var codec = msgpack.New()

// GetID for req - used for tracing and logging
func (r Request) GetID() string {
	return r.Meta["x-trace-id"]
}

// SetID for req - used for tracing and logging
func (r *Request) SetID(id string) interfaces.Request {
	if r.Meta == nil {
		r.Meta = map[string]string{}
	}
	r.Meta["x-trace-id"] = id
	return r
}

// GetTracerData for req
func (r Request) GetTracerData() map[string][]string {
	return r.TracerData
}

// SetTracerData for req
func (r *Request) SetTracerData(d map[string][]string) interfaces.Request {
	r.TracerData = d
	return r
}

// GetMeta for req
func (r Request) GetMeta() map[string]string {
	return r.Meta
}

// SetMeta for req
func (r *Request) SetMeta(m map[string]string) interfaces.Request {
	r.Meta = m
	return r
}

// GetTimeout for req
func (r Request) GetTimeout() *int {
	return r.Timeout
}

// SetTimeout for req
func (r *Request) SetTimeout(t int) interfaces.Request {
	r.Timeout = &t
	return r
}

// GetPath for req
func (r Request) GetPath() string {
	return r.Path
}

// SetPath for req
func (r *Request) SetPath(p string) interfaces.Request {
	r.Path = p
	return r
}

// GetParams for req
func (r *Request) GetParams() []byte {
	return r.Params
}

// SetParams for type
func (r *Request) SetParams(params interface{}) error {
	b, err := codec.Encode(params)
	r.Params = b
	return err
}

// ParseParams as type
func (r *Request) ParseParams(to interface{}) error {
	return codec.Decode(r.Params, to)
}
