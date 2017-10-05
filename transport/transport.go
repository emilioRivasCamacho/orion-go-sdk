package transport

// Options for nats
type Options struct {
	URL string
}

// Option type
type Option func(*Options)

// SetTransportURL for orion
func SetTransportURL(url string) Option {
	return func(o *Options) {
		o.URL = url
	}
}
