package interfaces

import "github.com/betit/orion/go/request"
import "github.com/betit/orion/go/logger"

// Codec interface
type Codec interface {
	Encode(...interface{}) ([]byte, error)
	Decode([]byte, ...interface{}) error
}

// Transport interface
type Transport interface {
	Listen(func())
	Publish(string, []byte) error
	Subscribe(string, string, func([]byte)) error
	Handle(string, string, func([]byte) []byte) error
	Request(string, []byte, int) ([]byte, error)
	Close()
}

// Tracer interface
type Tracer interface {
	Trace(*orequest.Request) func()
}

// Logger interface
type Logger = ologger.Logger
