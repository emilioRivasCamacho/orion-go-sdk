package logger

import (
	"encoding/json"
	"flag"
	"log"
	"strconv"
	"time"

	"github.com/betit/orion-go-sdk/env"
	"github.com/duythinht/gelf/client"
)

var (
	// DefaultParams that will be send to graylog
	DefaultParams = map[string]interface{}{}
	// Host for graylog
	Host = ""
	// Port for graylog
	Port    = ""
	verbose = flag.Bool("verbose", false, "Enable verbose (console) logging")
)

// Graylog logger
type Graylog struct {
	client  *client.Gelf
	service string
}

// Logger interface
type Logger interface {
	CreateMessage(message string) *Message
	Send(string)
}

func init() {
	flag.Parse()
	setVariables()
}

// New graylog logger
func New(serviceName string) *Graylog {
	port, err := strconv.Atoi(Port)
	if err != nil {
		log.Fatalf("Unable to parse graylog port %s", err)
	}

	c := client.New(client.Config{
		GraylogHost: Host,
		GraylogPort: port,
	})

	return &Graylog{
		client:  c,
		service: serviceName,
	}
}

// CreateMessage for graylog
func (g *Graylog) CreateMessage(message string) *Message {
	m := &Message{
		logger: g,
		args:   map[string]interface{}{},
	}
	m.args["host"] = g.service
	m.args["message"] = message
	m.args["timestamp"] = time.Now().Unix()

	return m
}

// Send message to graylog
func (g *Graylog) Send(m string) {
	g.client.Send(m)
}

// Message object
type Message struct {
	logger Logger
	args   map[string]interface{}
}

// SetLevel for message
func (m *Message) SetLevel(level int) *Message {
	m.args["level"] = level
	return m
}

// SetID for message
func (m *Message) SetID(id string) *Message {
	m.args["x-trace-id"] = id
	return m
}

// SetCode for error
func (m *Message) SetCode(code string) *Message {
	m.args["code"] = code
	return m
}

// SetMap will loop through the map and will add each entry as top level key
// for the message
func (m *Message) SetMap(p map[string]interface{}) *Message {
	for key, value := range p {
		m.args[key] = value
	}
	return m
}

// SetParams for message
func (m *Message) SetParams(p interface{}) *Message {
	b, _ := json.Marshal(p)
	m.args["raw"] = string(b)
	return m
}

// Send message
func (m *Message) Send() {
	for key, value := range DefaultParams {
		m.args[key] = value
	}

	b, _ := json.Marshal(m.args)
	data := string(b)

	if *verbose {
		log.Println(data)
	}

	m.logger.Send(data)
}

func setVariables() {
	if Host == "" {
		Host = env.Get("ORION_LOGGER_HOST", "127.0.0.1")
	}
	if Port == "" {
		Port = env.Get("ORION_LOGGER_PORT", "12201")
	}
}
