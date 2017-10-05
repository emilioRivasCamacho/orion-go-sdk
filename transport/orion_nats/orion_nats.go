package onats

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/betit/orion/go/transport"

	"github.com/nats-io/nats"
)

var (
	close = make(chan bool)
)

// NATSTransport object
type NATSTransport struct {
	options *transport.Options
	conn    *nats.Conn
}

// New nats transport
func New(options ...transport.Option) *NATSTransport {
	t := new(NATSTransport)

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

	return t
}

// Listen to nats
func (t *NATSTransport) Listen(callback func()) {
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
func (t *NATSTransport) Publish(topic string, data []byte) error {
	return t.conn.Publish(topic, data)
}

// Subscribe for topic
func (t *NATSTransport) Subscribe(topic string, group string, handler func([]byte)) error {
	_, err := t.conn.QueueSubscribe(topic, group, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	return err
}

// Handle path
func (t *NATSTransport) Handle(path string, group string, handler func([]byte) []byte) error {
	_, err := t.conn.QueueSubscribe(path, group, func(msg *nats.Msg) {
		res := handler(msg.Data)
		t.conn.Publish(msg.Reply, res)
	})
	return err
}

// Request path
func (t *NATSTransport) Request(path string, payload []byte, timeOut int) ([]byte, error) {

	msg, err := t.conn.Request(path, payload, time.Duration(timeOut)*time.Millisecond)
	var data []byte
	if msg != nil {
		data = msg.Data
	}

	return data, err
}

// Close connection
func (t *NATSTransport) Close() {
	t.conn.Close()
	go func() {
		close <- true
	}()
}
