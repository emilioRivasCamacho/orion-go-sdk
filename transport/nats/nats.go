package nats

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gig/orion-go-sdk/transport"

	"github.com/nats-io/go-nats"
)

// Conn is a NATS connection
type Conn = nats.Conn

// Transport object
type Transport struct {
	options      *transport.Options
	conn         *Conn
	close        chan struct{}
	closeHandler func(*nats.Conn)
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
	natsOpts := nats.Options{
		Url:            t.options.URL,
		AllowReconnect: true,
		MaxReconnect:   10,
		ReconnectWait:  1 * time.Second,
		Timeout:        2 * time.Second,
		PingInterval:   1 * time.Second,
	}
	t.conn, err = natsOpts.Connect()
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
	err := t.conn.Publish(topic, data)
	t.handleUnexpectedClose(err)
	return err
}

// Subscribe for topic
func (t *Transport) Subscribe(topic string, group string, handler func([]byte)) error {
	_, err := t.conn.QueueSubscribe(topic, group, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	t.handleUnexpectedClose(err)
	return err
}

// Handle path
func (t *Transport) Handle(path string, group string, handler func([]byte, func([]byte))) error {
	_, err := t.conn.QueueSubscribe(path, group, func(msg *nats.Msg) {
		handler(msg.Data, func(res []byte) {
			t.conn.Publish(msg.Reply, res)
		})
	})
	t.handleUnexpectedClose(err)
	return err
}

// Request path
func (t *Transport) Request(path string, payload []byte, timeOut int) ([]byte, error) {

	msg, err := t.conn.Request(path, payload, time.Duration(timeOut)*time.Millisecond)
	t.handleUnexpectedClose(err)
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

// OnClose adds a handler to NATS close event
func (t *Transport) OnClose(handler interface{}) {
	if callback, ok := handler.(func(*nats.Conn)); ok {
		t.closeHandler = callback
		t.conn.SetClosedHandler(callback)
	}
}

func (t *Transport) handleUnexpectedClose(err error) {
	if err == nats.ErrConnectionClosed {
		t.closeHandler(t.conn)
	}
}
