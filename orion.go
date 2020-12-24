package orion

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gig/orion-go-sdk/codec/msgpack"
	"github.com/gig/orion-go-sdk/env"
	oerror "github.com/gig/orion-go-sdk/error"
	"github.com/gig/orion-go-sdk/health"
	"github.com/gig/orion-go-sdk/health/checks"
	"github.com/gig/orion-go-sdk/interfaces"
	"github.com/gig/orion-go-sdk/logger"
	"github.com/gig/orion-go-sdk/response"
	"github.com/gig/orion-go-sdk/transport/nats"
	"github.com/panjf2000/ants"
	uuid "github.com/satori/go.uuid"
)

const defaultThreadPoolSize = 1

var (
	threadPoolSize = env.Get("THREADPOOL_SIZE", strconv.Itoa(defaultThreadPoolSize))
)

// Factory func type - the one that creates the req obj
type Factory = func() interfaces.Request

// Service for orion
type Service struct {
	ID                  string
	Name                string
	Timeout             int
	Codec               interfaces.Codec
	Transport           interfaces.Transport
	Logger              interfaces.Logger
	ThreadPool          *ants.PoolWithFunc
	HealthChecks        []health.Dependency
	StopHealthCheck     chan struct{}
	HTTPServer          *http.Server
	HTTPPort            int
	DisableHealthChecks bool
}

// DefaultServiceOptions setup
func DefaultServiceOptions(opt *Options) {
	if opt.HTTPPort == 0 {
		thePort, err := strconv.Atoi(env.Get("HTTP_SERVER_PORT", "9001"))
		if err != nil {
			panic(err)
		}
		opt.HTTPPort = thePort
	}
	opt.DisableHealthChecks = env.Truthy("DISABLE_HEALTH_CHECK")
}

// UniqueName for given name and unique id
func UniqueName(name string, uniqueID string) string {
	return name + "@" + uniqueID
}

// New orion service
func New(name string, options ...Option) *Service {
	opts := &Options{}

	DefaultServiceOptions(opts)

	for _, setter := range options {
		setter(opts)
	}

	if opts.Transport == nil {
		opts.Transport = nats.New()
	}

	// as for now, the codec will always be msgpack
	opts.Codec = msgpack.New()

	if opts.Logger == nil {
		opts.Logger = logger.New(name)
	}

	uid, _ := uuid.NewV4()

	poolSize, err := strconv.Atoi(threadPoolSize)

	if err != nil {
		poolSize = defaultThreadPoolSize
	}

	workerPool, err := ants.NewPoolWithFunc(poolSize, func(fn interface{}) {
		toCall := fn.(func())
		toCall()
	}, ants.WithNonblocking(false))

	if err != nil {
		panic(err)
	}

	s := &Service{
		ID:                  uid.String(),
		Name:                name,
		Timeout:             200,
		Codec:               opts.Codec,
		Transport:           opts.Transport,
		Logger:              opts.Logger,
		ThreadPool:          workerPool,
		HealthChecks:        make([]health.Dependency, 0),
		HTTPPort:            opts.HTTPPort,
		DisableHealthChecks: opts.DisableHealthChecks,
	}

	if !opts.DisableHealthChecks {
		s.RegisterHealthCheck(checks.NatsHealthcheck(opts.Transport))
	}

	return s
}

// Emit to services
func (s *Service) Emit(topic string, data interface{}) error {
	msg, err := s.Codec.Encode(data)
	if err != nil {
		return err
	}
	return s.Transport.Publish(topic, msg)
}

// On service emit
func (s *Service) On(topic string, handler func([]byte)) {
	subject := fmt.Sprintf("%s:%s", s.Name, topic)
	s.Transport.Subscribe(subject, s.Name, handler)
}

// SubscribeForRawMsg is like service.On except that it receives the raw messages
// specific for the transport protocol instead of the message payload
func (s *Service) SubscribeForRawMsg(topic string, handler func(interface{})) {
	subject := fmt.Sprintf("%s:%s", s.Name, topic)
	s.Transport.SubscribeForRawMsg(subject, s.Name, handler)
}

// Decode bytes to passed interface
func (s *Service) Decode(data []byte, to interface{}) error {
	return s.Codec.Decode(data, &to)
}

// HandleWithoutLogging works the same as Handle but with disabled logging
func (s *Service) HandleWithoutLogging(path string, handler interface{}, factory Factory) {
	s.handle(path, logger.NONE, handler, factory)
}

// HandleWithCustomLogLevel works the same as Handle but it lets you set the log level
func (s *Service) HandleWithCustomLogLevel(path string, logLevel int, handler interface{}, factory Factory) {
	s.handle(path, logLevel, handler, factory)
}

// Handle has enabled logging. What that means is when the request
// arrives the service will log the request including the raw params. Once the
// response is returned, the service will check for error and if there is such,
// the error will be logged
func (s *Service) Handle(path string, handler interface{}, factory Factory) {
	s.handle(path, logger.INFO, handler, factory)
}

func (s *Service) handle(path string, logLevel int, handler interface{}, factory Factory) {
	route := s.getRouteFromPath(path)

	method := reflect.ValueOf(handler)
	s.checkHandler(method)

	s.Transport.Handle(route, s.Name, func(data []byte, reply func([]byte)) {
		toProcess := func() {
			req := factory()

			err := s.Codec.Decode(data, req)
			req.SetError(err)

			s.logRequest(err, req, logLevel)

			if err != nil {
				panic(err)
			}

			res := method.Call([]reflect.Value{reflect.ValueOf(req)})[0].Interface()

			s.logResponse(req, res, logLevel)

			b, err := s.Codec.Encode(res)
			if err != nil {
				log.Fatal(err)
			}

			reply(b)
		}

		s.ThreadPool.Invoke(toProcess)
	})
}

func (s *Service) RegisterHealthCheck(check *health.Dependency) {
	// We store the original check function
	realCheck := check.CheckIsWorking

	// And we change the original check function with another one which times out if
	// the check delays too much
	check.CheckIsWorking = health.WrapHealthCheckWithATimeout(check.Name, check.Timeout, realCheck)
	s.HealthChecks = append(s.HealthChecks, *check)

	// Then we restore the original object
	check.CheckIsWorking = realCheck
}

// Call orion service
func (s *Service) Call(req interfaces.Request, raw interface{}) {
	res, ok := raw.(interfaces.Response)
	checkResponseCast(ok)

	encoded, err := s.Codec.Encode(req)
	if err != nil {
		res.SetError(oerror.New("ORION_ENCODE").SetMessage(err.Error()).SetLineOfCode(oerror.GenerateLOC(1)))
		return
	}

	path := replaceOmitEmpty(req.GetPath(), "/", ".")
	b, err := s.Transport.Request(path, encoded, s.getTimeout(req))
	if err != nil {
		res.SetError(oerror.New("ORION_TRANSPORT").SetMessage(err.Error()).SetLineOfCode(oerror.GenerateLOC(1)))
		return
	}

	err = s.Codec.Decode(b, res)
	if err != nil {
		res.SetError(oerror.New("ORION_DECODE").SetMessage(err.Error()).SetLineOfCode(oerror.GenerateLOC(1)))

		s.Logger.
			CreateMessage("ORION_DECODE " + req.GetPath()).
			SetLevel(logger.ERROR).
			SetID(req.GetID()).
			SetMap(map[string]interface{}{
				"error": res.GetError(),
			}).
			SetLineOfCode(oerror.GenerateLOC(1)).
			Send()

		return
	}
}

type responseWithAnyPayload struct {
	response.Response
	Payload interface{} `msgpack:"payload"`
}

func (s *Service) loopOverHealthChecks() {
	s.StopHealthCheck = make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// TODO: What to do with the error?
				fmt.Println(r)
			}
		}()
		health.LoopOverHealthChecks(s.HealthChecks, s.StopHealthCheck)
	}()
}

// Listen to the transport protocol
func (s *Service) Listen(callback func()) {
	if !s.DisableHealthChecks {
		s.loopOverHealthChecks()
		s.HTTPServer = health.StartHTTPServer(":" + strconv.Itoa(s.HTTPPort))
	}

	s.Transport.Listen(callback)
}

// Close the transport protocol
func (s *Service) Close() {
	if s.StopHealthCheck != nil {
		s.StopHealthCheck <- struct{}{}
		s.StopHealthCheck = nil
	}

	if s.HTTPServer != nil {
		health.CloseHTTPServer(s.HTTPServer)
		s.HTTPServer = nil
	}

	s.Transport.Close()
}

// OnClose adds a handler to a transport connection closed event
func (s *Service) OnClose(handler func()) {
	s.Transport.OnClose(func(*nats.Conn) {
		handler()
	})
}

// String return the name and the id of the service
func (s *Service) String() string {
	return fmt.Sprintf("%s-%s", s.Name, s.ID)
}

func (s *Service) logRequest(err error, raw interface{}, logLevel int) {
	if logLevel != logger.NONE {

		req, ok := raw.(interfaces.Request)
		checkRequestCast(ok)

		v := reflect.ValueOf(raw).Elem().FieldByName("Params").Interface()

		var out interface{}
		var in interface{}

		if _, ok = v.([]byte); ok {
			s.Decode(req.GetParams(), &in)
			t, _ := json.Marshal(in)
			out = string(t)
		} else {
			out = v
		}

		params := map[string]interface{}{
			"params": out,
			"meta":   req.GetMeta(),
		}

		level := logLevel
		if err != nil {
			level = logger.ERROR
			params["error"] = err.Error()
		}

		s.Logger.
			CreateMessage(req.GetPath()).
			SetLevel(level).
			SetID(req.GetID()).
			SetMap(params).
			Send()
	}
}

func (s Service) logResponse(rawReq, rawRes interface{}, logLevel int) {
	if logLevel != logger.NONE {

		req, ok := rawReq.(interfaces.Request)
		checkResponseCast(ok)

		res, ok := rawRes.(interfaces.Response)
		checkResponseCast(ok)

		err := res.GetError()
		if err != nil {
			s.Logger.
				CreateMessage(req.GetPath()).
				SetLevel(logLevel).
				SetID(req.GetID()).
				SetLineOfCode(err.LOC).
				SetParams(err).
				Send()
		}
	}
}

func (s Service) getTimeout(req interfaces.Request) int {
	t := req.GetTimeout()
	if t != nil {
		return *t
	}
	return s.Timeout
}

func (s Service) checkHandler(method reflect.Value) {
	if method.Type().NumIn() != 1 {
		log.Fatal(errors.New("handler methods must have one argument"))
	}

	if method.Type().NumOut() != 1 {
		log.Fatal(errors.New("handler methods must have one return value"))
	}
}

func (s Service) getRouteFromPath(path string) string {
	parts := strings.Split(path, "/")
	switch len(parts) {
	case 1:
		return fmt.Sprintf("%s.%s", s.Name, path)
	case 2:
		return fmt.Sprintf("%s.%s", parts[0], parts[1])
	default:
		log.Fatal(errors.New("handler path cannot contain more than one slash"))
		return fmt.Sprintf("%s.%s", s.Name, path)
	}
}

func replaceOmitEmpty(str string, split string, join string) string {
	var r []string
	for _, str := range strings.Split(str, split) {
		if str != "" {
			r = append(r, str)
		}
	}
	return strings.Join(r, join)
}

func checkRequestCast(ok bool) {
	if !ok {
		log.Fatal("Request does not implement interfaces.Request")
	}
}

func checkResponseCast(ok bool) {
	if !ok {
		log.Fatal("Response does not implement interfaces.Response")
	}
}
