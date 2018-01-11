package logger

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/betit/orion-go-sdk/env"
	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/duythinht/gelf/client"
	logging "github.com/op/go-logging"
)

var (
	// DefaultParams that will be send to graylog
	DefaultParams = map[string]interface{}{}
	// Host for graylog
	Host = ""
	// Port for graylog
	Port    = ""
	verbose *bool
	log     *logging.Logger
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
	setVariables()
	initConsoleLogger()
}

// New graylog logger
func New(serviceName string) *Graylog {
	parseFlags()

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
	m.args["params"] = string(b)
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
		m.log(data)
	}

	m.logger.Send(data)
}

func (m *Message) log(data string) {

	switch m.args["level"] {
	case EMERGENCY:
		log.Critical("Emergency %s", data)
	case CRITICAL:
		log.Critical("Critical %s", data)
	case ERROR:
		log.Error("Error %s", data)
	case ALERT:
		log.Warning("Alert %s", data)
	case WARNING:
		log.Warning("Warning %s", data)
	case NOTICE:
		log.Notice("Notice %s", data)
	case INFO:
		log.Debug("Info %s", data)
	case DEBUG:
		log.Info("Debug %s", data)
	default:
		log.Info("Debug %s", data)
	}

}
func (message *Message) SetLineOfCode(code oerror.LineOfCode) *Message {
	message.args["LOC"] = code.File + ":" + strconv.Itoa(code.Line)

	return message
}

func setVariables() {
	if Host == "" {
		Host = env.Get("ORION_LOGGER_HOST", "127.0.0.1")
	}
	if Port == "" {
		Port = env.Get("ORION_LOGGER_PORT", "12201")
	}
}

func parseFlags() {
	f := flag.Lookup("verbose")
	if f == nil {
		verbose = flag.Bool("verbose", false, "Enable verbose (console) logging")
		flag.Parse()
	} else {
		b, _ := strconv.ParseBool(f.Value.String())
		verbose = &b
	}
}

func initConsoleLogger() {
	log = logging.MustGetLogger("Orion")
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} ▶ %{color:reset} %{message}`,
	)
	logging.SetBackend(logging.NewBackendFormatter(backend, format))
}
