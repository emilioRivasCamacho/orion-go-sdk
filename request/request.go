package request

import (
	"strconv"
	"time"

	"github.com/gig/orion-go-sdk/codec/msgpack"
	"github.com/gig/orion-go-sdk/interfaces"
	uuid "github.com/satori/go.uuid"
)

// Meta type for req
type Meta map[string]string

// TracerData type for req
type TracerData map[string][]string

// Request object
// swagger:ignore
type Request struct {
	// Empty json tags because we need to omit those fields when generating the docs
	// and we do not plan to support json
	TracerData TracerData `json:"-" msgpack:"tracerData"`
	Path       string     `json:"-" msgpack:"path"`
	Params     []byte     `json:"-" msgpack:"params"`
	Meta       Meta       `json:"-" msgpack:"meta"`
	Timeout    *int       `json:"-" msgpack:"timeout"`
	Error      error      `json:"-" msgpack:",omitempty"`
}

var codec = msgpack.New()

// New request
func New() *Request {
	uid, _ := uuid.NewV4()
	return &Request{
		Meta: map[string]string{
			"x-trace-id": uid.String(), // will be overriden by tracer is tracing is enabled
		},
		TracerData: map[string][]string{},
	}
}

// Increases by one the propagation level. If missing, it sets to one.
func increasePropagationLevel(r interfaces.Request) {
	propagationLevelS, ok := r.GetMeta()["propagation"]

	if !ok {
		propagationLevelS = "0"
	}

	propagationLevel, _ := strconv.Atoi(propagationLevelS)
	propagationLevel += 1
	r.GetMeta()["propagation"] = strconv.Itoa(propagationLevel)
}

// Merge the meta data
// Needed for cross service communication
func Merge(from, to interfaces.Request) {
	to.SetMeta(from.GetMeta())
	to.SetTracerData(from.GetTracerData())
	increasePropagationLevel(to)
}

// GetID for req - used for tracing and logging
func (r Request) GetID() string {
	return r.GetMetaProp("x-trace-id")
}

// SetID for req - used for tracing and logging
func (r *Request) SetID(id string) interfaces.Request {
	return r.SetMetaProp("x-trace-id", id)
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
	if r.Meta == nil {
		r.Meta = make(Meta)
	}
	for key, value := range m {
		r.Meta[key] = value
	}
	return r
}

// GetMetaProp for req
func (r Request) GetMetaProp(key string) string {
	return r.Meta[key]
}

// SetMetaProp for req
func (r *Request) SetMetaProp(key, value string) interfaces.Request {
	if r.Meta == nil {
		r.Meta = map[string]string{}
	}
	r.Meta[key] = value
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

// SetTimeoutDuration for req based on milliseconds
func (r *Request) SetTimeoutDuration(t time.Duration) interfaces.Request {
	t2 := int(t / time.Millisecond)
	r.Timeout = &t2
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
func (r Request) GetParams() []byte {
	return r.Params
}

// SetParams for type
func (r *Request) SetParams(params interface{}) error {
	b, err := codec.Encode(params)
	r.Params = b
	return err
}

// ParseParams as type
func (r Request) ParseParams(to interface{}) error {
	return codec.Decode(r.Params, to)
}

// SetError that is returned when decoding the bytes (raw req)
func (r *Request) SetError(err error) interfaces.Request {
	r.Error = err
	return r
}
