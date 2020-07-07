package orion

import "github.com/gig/orion-go-sdk/interfaces"

// client-service

// Options object
type Options struct {
	Codec               interfaces.Codec
	Transport           interfaces.Transport
	Logger              interfaces.Logger
	DisableHealthChecks bool
	HTTPPort            int
	Register            interfaces.Register
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
