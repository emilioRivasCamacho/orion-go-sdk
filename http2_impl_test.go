package orion

import (
	"testing"

	jsoncodec "github.com/gig/orion-go-sdk/codec/json"

	"github.com/stretchr/testify/assert"

	"github.com/gig/orion-go-sdk/transport/http2"
)

func TestRegisterTraefik(t *testing.T) {
	http2Transport := http2.New()
	err := http2Transport.Register("test_service", "test_service@1", []string{"prefix1"})
	assert.Nil(t, err, "err must be nil")
}

func TestListenTraefik(t *testing.T) {
	http2Transport := http2.New()
	wait := make(chan struct{})
	go http2Transport.Listen(func() {
		wait <- struct{}{}
	})
	<-wait
}

func TestPublishTraefik(t *testing.T) {
	http2Transport := http2.New()

	codec := jsoncodec.JSONCodec{}
	j := make(map[string]interface{})
	err := codec.Decode([]byte("{\"params\": {\"nested\":{\"time\":\"Thu Jul 02 2020 13:40:08 GMT+0000 (GMT)\"}}, \"meta\": {}}"), &j)

	err = http2Transport.Publish("examples:node", []byte("{\"message\": {\"nested\":{\"time\":\"Thu Jul 02 2020 13:40:08 GMT+0000 (GMT)\"}}}"))
	assert.Nil(t, err, "err must be nil")
}

func TestSubscribe(t *testing.T) {

}
