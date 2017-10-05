package omsgpack

import (
	"github.com/vmihailenco/msgpack"
)

// MSGPack object
type MSGPack struct{}

// New msgpack coded
func New() *MSGPack {
	return new(MSGPack)
}

// Encode values
func (c *MSGPack) Encode(v ...interface{}) ([]byte, error) {
	return msgpack.Marshal(v...)
}

// Decode values
func (c *MSGPack) Decode(b []byte, v ...interface{}) error {
	return msgpack.Unmarshal(b, v...)
}
