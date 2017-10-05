package msgpack

import (
	msgp "github.com/vmihailenco/msgpack"
)

// MSGPack object
type MSGPack struct{}

// New msgpack coded
func New() *MSGPack {
	return new(MSGPack)
}

// Encode values
func (c *MSGPack) Encode(v ...interface{}) ([]byte, error) {
	return msgp.Marshal(v...)
}

// Decode values
func (c *MSGPack) Decode(b []byte, v ...interface{}) error {
	return msgp.Unmarshal(b, v...)
}
