package transport

// Options for nats
type Options struct {
	URL            string
	Http2Port      int
	PoolThreadSize int
}

// Option type
type Option func(*Options)

// SetTransportURL for orion
func SetTransportURL(url string) Option {
	return func(o *Options) {
		o.URL = url
	}
}

// SetTransportPort for orion
func SetTransportPort(port int) Option {
	return func(o *Options) {
		o.Http2Port = port
	}
}
