package jsoncodec

import (
	"encoding/json"
	"log"
)

// JSONCodec object
type JSONCodec struct{}

// New JSONCodec codec
func New() *JSONCodec {
	return new(JSONCodec)
}

// Encode values
func (j *JSONCodec) Encode(v ...interface{}) ([]byte, error) {
	if len(v) != 1 {
		log.Panic("this encoder only supports single item encoding")
	}
	return json.Marshal(v[0])
}

// Decode values
// TODO: We might want to change the interface of the decoder to accept only a single item.
func (j *JSONCodec) Decode(b []byte, v ...interface{}) error {
	if len(v) != 1 {
		log.Panic("this encoder only supports single item encoding")
	}
	return json.Unmarshal(b, v[0])
}
