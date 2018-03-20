package orion

import "github.com/gig/orion-go-sdk/interfaces"

// client-service

// Options object
type Options struct {
	Codec                 interfaces.Codec
	Transport             interfaces.Transport
	Tracer                interfaces.Tracer
	Logger                interfaces.Logger
	RegisterToWatchdog    bool
	EnableStatusEndpoints bool
	WatchdogServiceName   string
}

// Option type
type Option func(*Options)

// SetCodec for orion
func SetCodec(codec interfaces.Codec) Option {
	return func(o *Options) {
		o.Codec = codec
	}
}

// SetTransport for orion
func SetTransport(transport interfaces.Transport) Option {
	return func(o *Options) {
		o.Transport = transport
	}
}

// SetTracer for orion
func SetTracer(tracer interfaces.Tracer) Option {
	return func(o *Options) {
		o.Tracer = tracer
	}
}
