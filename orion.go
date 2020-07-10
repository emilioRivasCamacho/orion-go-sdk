package orion

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	jsoncodec "github.com/gig/orion-go-sdk/codec/json"

	"github.com/gig/orion-go-sdk/transport"
	"github.com/phayes/freeport"

	"github.com/gig/orion-go-sdk/transport/http2"

	"github.com/gig/orion-go-sdk/env"
	oerror "github.com/gig/orion-go-sdk/error"
	"github.com/gig/orion-go-sdk/health"
	"github.com/gig/orion-go-sdk/interfaces"
	"github.com/gig/orion-go-sdk/logger"
	"github.com/gig/orion-go-sdk/response"
	uuid "github.com/satori/go.uuid"
)

const defaultThreadPoolSize = 1

var (
	verbose        = env.Truthy("VERBOSE")
	threadPoolSize = env.Get("THREADPOOL_SIZE", strconv.Itoa(defaultThreadPoolSize))
)

// Factory func type - the one that creates the req obj
type Factory = func() interfaces.Request

// Service for orion
type Service struct {
	ID                  string
	Name                string
	InstanceName        string
	Timeout             int
	Codec               interfaces.Codec
	Transport           interfaces.Transport
	Register            interfaces.Register
	Logger              interfaces.Logger
	HealthChecks        []health.Dependency
	StopHealthCheck     chan struct{}
	HTTPPort            int
	DisableHealthChecks bool
	Prefixes            map[string]struct{}
}

// DefaultServiceOptions setup
func DefaultServiceOptions(opt *Options) {
	if opt.HTTPPort == 0 {
		thePort, err := strconv.Atoi(env.Get("HTTP_SERVER_PORT", "0"))
		if err != nil {
			panic(err)
		}
		opt.HTTPPort = thePort
	}
	opt.DisableHealthChecks = env.Truthy("DISABLE_HEALTH_CHECK")
}

// New orion service
func New(name string, options ...Option) *Service {
	opts := &Options{}

	DefaultServiceOptions(opts)

	opts.Codec = jsoncodec.New()

	for _, setter := range options {
		setter(opts)
	}

	poolSize, err := strconv.Atoi(threadPoolSize)

	if err != nil {
		poolSize = defaultThreadPoolSize
	}

	if opts.Transport == nil {
		var port int

		if opts.HTTPPort == 0 {
			port, err = freeport.GetFreePort()

			if err != nil {
				log.Fatal(err)
			}
			opts.HTTPPort = port
		}

		t := http2.New(func(options *transport.Options) {
			options.Http2Port = port
			options.URL = "host.docker.internal"
			options.PoolThreadSize = poolSize
			options.Codec = opts.Codec
		})

		opts.Transport = t

		if opts.Register == nil {
			opts.Register = t
		}
	}

	if opts.Logger == nil {
		opts.Logger = logger.New(name, verbose)
	}

	uid, _ := uuid.NewV4()

	s := &Service{
		ID:                  uid.String(),
		Name:                name,
		InstanceName:        name + "@" + uid.String(),
		Timeout:             200,
		Codec:               opts.Codec,
		Transport:           opts.Transport,
		Logger:              opts.Logger,
		HealthChecks:        make([]health.Dependency, 0),
		HTTPPort:            opts.HTTPPort,
		DisableHealthChecks: opts.DisableHealthChecks,
		Register:            opts.Register,
		Prefixes:            make(map[string]struct{}),
	}

	return s
}

// Emit to services
func (s *Service) Emit(topic string, data interface{}) error {
	route := "/" + strings.ReplaceAll(topic, ":", "/")

	msg, err := s.Codec.Encode(data)
	if err != nil {
		return err
	}

	return s.Transport.Publish(route, msg)
}

// On service emit
func (s *Service) On(topic string, handler func([]byte)) {
	s.Transport.Subscribe(topic, s.Name, handler)
}

// SubscribeForRawMsg is like service.On except that it receives the raw messages
// specific for the transport protocol instead of the message payload
func (s *Service) SubscribeForRawMsg(topic string, handler func(interface{})) {
	s.Transport.SubscribeForRawMsg(topic, s.Name, handler)
}

// Decode bytes to passed interface
func (s *Service) Decode(data []byte, to interface{}) error {
	return s.Codec.Decode(data, &to)
}

// HandleWithoutLogging enabled
func (s *Service) HandleWithoutLogging(path string, handler interface{}, factory Factory) {
	s.handle(path, false, handler, factory)
}

// Handle has enabled logging. What that means is when the request
// arrives the service will log the request including the raw params. Once the
// response is returned, the service will check for error and if there is such,
// the error will be logged
func (s *Service) Handle(path string, handler interface{}, factory Factory) {
	s.handle(path, true, handler, factory)
}

func (s *Service) extractGroup(path string) (string, string, bool) {
	if path[0] == '/' {
		return s.Name, path[1:], false
	}

	split := strings.Split(path, "/")

	if len(split) == 1 {
		return s.Name, path, false
	} else {
		return split[0], strings.Join(split[1:], "/"), true
	}
}

func (s *Service) handle(path string, logging bool, handler interface{}, factory Factory) {
	method := reflect.ValueOf(handler)
	s.checkHandler(method)

	group, route, isPrefix := s.extractGroup(path)

	if isPrefix {
		s.Prefixes[group] = struct{}{}
	}

	s.Transport.Handle(route, group, func(data []byte, reply func(res interfaces.Response)) {
		req := factory()

		err := s.Codec.Decode(data, req)
		req.SetError(err)

		s.logRequest(err, req, logging)

		if err != nil {
			panic(err)
		}

		res := method.Call([]reflect.Value{reflect.ValueOf(req)})[0].Interface()

		s.logResponse(req, res, logging)

		reply(res.(interfaces.Response))
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

	path := req.GetPath()
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
	if s.Register != nil {
		var prefixes []string

		for k, _ := range s.Prefixes {
			prefixes = append(prefixes, k)
		}

		err := s.Register.Register(s.Name, s.Name+"@"+s.ID, prefixes)

		if err != nil {
			log.Panic(err)
		}
	}

	if !s.DisableHealthChecks {
		s.loopOverHealthChecks()
		health.InstallHealthcheck(s.Transport, "healthcheck", s.InstanceName)
	}

	s.Transport.Listen(callback)
}

// Close the transport protocol
func (s *Service) Close() {
	if s.StopHealthCheck != nil {
		s.StopHealthCheck <- struct{}{}
		s.StopHealthCheck = nil
	}

	s.Transport.Close()
}

// OnClose adds a handler to a transport connection closed event
func (s *Service) OnClose(handler func()) {
	s.Transport.OnClose(func() {
		handler()
	})
}

// String return the name and the id of the service
func (s *Service) String() string {
	return fmt.Sprintf("%s-%s", s.Name, s.ID)
}

func (s *Service) logRequest(err error, raw interface{}, logging bool) {
	if logging {

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

		level := logger.INFO
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

func (s Service) logResponse(rawReq, rawRes interface{}, logging bool) {
	if logging {

		req, ok := rawReq.(interfaces.Request)
		checkResponseCast(ok)

		res, ok := rawRes.(interfaces.Response)
		checkResponseCast(ok)

		err := res.GetError()
		if err != nil {
			s.Logger.
				CreateMessage(req.GetPath()).
				SetLevel(logger.ERROR).
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
