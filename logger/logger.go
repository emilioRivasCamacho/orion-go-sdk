package logger

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/gig/gelf/client"
	"github.com/gig/orion-go-sdk/env"
	oerror "github.com/gig/orion-go-sdk/error"
	logging "github.com/op/go-logging"
)

var (
	// DefaultParams that will be send to graylog
	DefaultParams = map[string]interface{}{}
	// GraylogHost env var
	GraylogHost = ""
	// GraylogPort env var
	GraylogPort = ""
	host        = env.Get("HOST", "")
	minLogLevel = env.Get("ORION_LOGGER_LEVEL", "info")
	verbose     = env.Truthy("VERBOSE")
	stdoutOnly  = env.Truthy("LOGGER_STDOUT_ONLY")
)

// Graylog logger
type Graylog struct {
	log     *logging.Logger
	client  *client.Gelf
	service string
}

// Logger interface
type Logger interface {
	CreateMessage(message string) *Message
	Send(int, string)
}

func init() {
	setVariables()
}

// New graylog logger
func New(serviceName string) *Graylog {
	log := initConsoleLogger(serviceName)

	port, err := strconv.Atoi(GraylogPort)
	if err != nil {
		log.Fatalf("Unable to parse graylog port %s", err)
	}

	c := client.New(client.Config{
		GraylogHost: GraylogHost,
		GraylogPort: port,
	})

	return &Graylog{
		log:     log,
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
	m.args["vm_host"] = host
	m.args["host"] = g.service
	m.args["message"] = message
	m.args["timestamp"] = float64(time.Now().UnixNano()) / float64(time.Second)
	m.args["level"] = INFO

	return m
}

// Send message to graylog
func (g *Graylog) Send(level int, m string) {
	if verbose || stdoutOnly {
		g.stdout(level, m)
	}
	if stdoutOnly {
		return
	}
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

// ShouldSkip checks the log level and do not log the messag if the level is too low
func (m *Message) ShouldSkip() bool {
	level := levelToNumber(minLogLevel)
	msgLevel := m.args["level"].(int)

	return level < msgLevel
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

func (message *Message) SetLineOfCode(code oerror.LineOfCode) *Message {
	message.args["LOC"] = code.File + ":" + strconv.Itoa(code.Line)

	return message
}

// Send message
func (m *Message) Send() {
	if m.ShouldSkip() {
		return
	}

	for key, value := range DefaultParams {
		m.args[key] = value
	}

	b, _ := json.Marshal(m.args)
	data := string(b)

	if m.args["level"] == nil {
		m.args["level"] = DEBUG
	}

	m.logger.Send(m.args["level"].(int), data)
}

func (g *Graylog) stdout(level int, data string) {

	switch level {
	case EMERGENCY:
		g.log.Critical("Emergency %s", data)
	case CRITICAL:
		g.log.Critical("Critical %s", data)
	case ERROR:
		g.log.Error("Error %s", data)
	case ALERT:
		g.log.Warning("Alert %s", data)
	case WARNING:
		g.log.Warning("Warning %s", data)
	case NOTICE:
		g.log.Notice("Notice %s", data)
	case INFO:
		g.log.Debug("Info %s", data)
	case DEBUG:
		g.log.Info("Debug %s", data)
	default:
		g.log.Info("Debug %s", data)
	}
}

func setVariables() {
	if GraylogHost == "" {
		GraylogHost = env.Get("ORION_LOGGER_HOST", "127.0.0.1")
	}
	if GraylogPort == "" {
		GraylogPort = env.Get("ORION_LOGGER_PORT", "12201")
	}
}

func initConsoleLogger(serviceName string) *logging.Logger {
	log := logging.MustGetLogger(serviceName)
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} | %{module} ▶ %{color:reset} %{message}`,
	)
	logging.SetBackend(logging.NewBackendFormatter(backend, format))
	return log
}
