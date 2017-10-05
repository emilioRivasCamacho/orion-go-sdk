package nats

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/betit/orion-go-sdk/transport"

	n "github.com/nats-io/nats"
)

var (
	close = make(chan bool)
)

// Transport object
type Transport struct {
	options *transport.Options
	conn    *n.Conn
}

// New nats transport
func New(options ...transport.Option) *Transport {
	t := new(Transport)

	natsURL := os.Getenv("NATS")
	if natsURL == "" {
		natsURL = n.DefaultURL
	}

	t.options = &transport.Options{
		URL: natsURL,
	}

	for _, setter := range options {
		setter(t.options)
	}

	var err error
	t.conn, err = n.Connect(t.options.URL)
	if err != nil {
		log.Fatal(err)
	}

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
		close <- true
	}()
	<-close
	os.Exit(0)
}

// Publish to topic
func (t *Transport) Publish(topic string, data []byte) error {
	return t.conn.Publish(topic, data)
}

// Subscribe for topic
func (t *Transport) Subscribe(topic string, group string, handler func([]byte)) error {
	_, err := t.conn.QueueSubscribe(topic, group, func(msg *n.Msg) {
		handler(msg.Data)
	})
	return err
}

// Handle path
func (t *Transport) Handle(path string, group string, handler func([]byte) []byte) error {
	_, err := t.conn.QueueSubscribe(path, group, func(msg *n.Msg) {
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
	t.conn.Close()
	go func() {
		close <- true
	}()
}
