package nats

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/betit/orion-go-sdk/transport"

	"github.com/nats-io/go-nats"
)

// Transport object
type Transport struct {
	options *transport.Options
	conn    *nats.Conn
	close   chan struct{}
}

// New returns client for NATS messaging
func New(options ...transport.Option) *Transport {
	t := new(Transport)

	natsURL := os.Getenv("NATS")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	t.options = &transport.Options{
		URL: natsURL,
	}

	for _, setter := range options {
		setter(t.options)
	}

	var err error
	t.conn, err = nats.Connect(t.options.URL)
	if err != nil {
		log.Fatal(err)
	}

	t.close = make(chan struct{})
	return t
}

// Listen to nats
func (t *Transport) Listen(callback func()) {
	t.conn.Flush()
	callback()

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		t.close <- struct{}{}
	}()
	<-t.close
	t.conn.Close()
}

// Publish to topic
func (t *Transport) Publish(topic string, data []byte) error {
	return t.conn.Publish(topic, data)
}

// Subscribe for topic
func (t *Transport) Subscribe(topic string, group string, handler func([]byte)) error {
	_, err := t.conn.QueueSubscribe(topic, group, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	return err
}

// Handle path
func (t *Transport) Handle(path string, group string, handler func([]byte) []byte) error {
	_, err := t.conn.QueueSubscribe(path, group, func(msg *nats.Msg) {
		res := handler(msg.Data)
		t.conn.Publish(msg.Reply, res)
	})
	return err
}

// Request path
func (t *Transport) Request(path string, payload []byte, timeOut int) ([]byte, error) {

	msg, err := t.conn.Request(path, payload, time.Duration(timeOut)*time.Millisecond)
	var data []byte
	if msg != nil {
		data = msg.Data
	}

	return data, err
}

// Close connection
func (t *Transport) Close() {
	go func() {
		t.close <- struct{}{}
	}()
}
