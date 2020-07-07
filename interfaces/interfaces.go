package interfaces

import (
	"time"

	oerror "github.com/gig/orion-go-sdk/error"
	"github.com/gig/orion-go-sdk/logger"
)

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
	SubscribeForRawMsg(string, string, func(interface{})) error
	Handle(string, string, func([]byte, func([]byte))) error
	Request(string, []byte, int) ([]byte, error)
	Close()
	IsOpen() bool
	OnClose(interface{})
}

// Response interface
type Response interface {
	GetError() *oerror.Error
	SetError(*oerror.Error) Response
	ParsePayload(interface{}) error
	SetPayload(interface{}) error
}

// Request interface
type Request interface {
	GetPath() string
	SetPath(string) Request
	GetID() string
	SetID(string) Request
	GetTimeout() *int
	SetTimeout(int) Request
	SetTimeoutDuration(duration time.Duration) Request
	GetMeta() map[string]string
	SetMeta(map[string]string) Request
	GetMetaProp(key string) string
	SetMetaProp(key, value string) Request
	GetParams() []byte
	ParseParams(interface{}) error
	SetParams(interface{}) error
	SetError(error) Request
}

// Register interface
type Register interface {
	Register(serviceName string, instanceName string, prefixList []string) error
}

// Logger interface
type Logger = logger.Logger
