package orion

import (
	"testing"

	"github.com/gig/orion-go-sdk/transport/kafka"
	"github.com/stretchr/testify/assert"
)

func TestKafkaPubSub(t *testing.T) {
	done := make(chan string)
	expected := "foo bar bazz"

	s := New("bar", SetTransport(kafka.New()))

	s.On("foobarbaz", func(b []byte) {
		s.Close()
		res := ""
		s.Decode(b, &res)
		done <- res
	})

	s.Listen(func() {
		s.Emit("bar:foobarbaz", expected)
	})

	res := <-done
	assert.Equal(t, expected, res)
}
