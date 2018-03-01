package orion

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"time"

	"github.com/betit/orion-go-sdk/codec/msgpack"
	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/betit/orion-go-sdk/health"
	"github.com/betit/orion-go-sdk/interfaces"
	"github.com/betit/orion-go-sdk/logger"
	"github.com/betit/orion-go-sdk/response"
	"github.com/betit/orion-go-sdk/tracer"
	"github.com/betit/orion-go-sdk/transport/nats"
	uuid "github.com/satori/go.uuid"
)

// Factory func type - the one that creates the req obj
type Factory = func() interfaces.Request

// Service for orion
type Service struct {
	ID                    string
	Name                  string
	Timeout               int
	Codec                 interfaces.Codec
	Transport             interfaces.Transport
	Tracer                interfaces.Tracer
	Logger                interfaces.Logger
	Endpoints             []string
	RegisterToWatchdog    bool
	EnableStatusEndpoints bool
	WatchdogServiceName   string
	closeWatchdogChannel  chan bool
	HealthChecks          map[string]health.Dependency
}

func DefaultServiceOptions(opt *Options) {
	opt.RegisterToWatchdog = true
	opt.EnableStatusEndpoints = true
	opt.WatchdogServiceName = health.DefaultWatchdogServiceName()
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

	if opts.Tracer == nil {
		opts.Tracer = tracer.New(name)
	}

	if opts.Logger == nil {
		opts.Logger = logger.New(name)
	}

	uid, _ := uuid.NewV4()

	s := &Service{
		ID:                    uid.String(),
		Name:                  name,
		Timeout:               200,
		Codec:                 opts.Codec,
		Transport:             opts.Transport,
		Tracer:                opts.Tracer,
		Logger:                opts.Logger,
		HealthChecks:          make(map[string]health.Dependency),
		RegisterToWatchdog:    opts.RegisterToWatchdog,
		EnableStatusEndpoints: opts.EnableStatusEndpoints,
		WatchdogServiceName:   opts.WatchdogServiceName,
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

// Decode bytes to passed interface
func (s *Service) Decode(data []byte, to interface{}) error {
	return s.Codec.Decode(data, &to)
}

// HandleWithoutLogging enabled
func (s *Service) HandleWithoutLogging(path string, handler interface{}, factory Factory) {
	logging := false
	s.handle(path, logging, handler, factory)
}

// Handle has enabled logging. What that means is when the request
// arrives the service will log the request including the raw params. Once the
// response is returned, the service will check for error and if there is such,
// the error will be logged
func (s *Service) Handle(path string, handler interface{}, factory Factory) {
	logging := true
	s.handle(path, logging, handler, factory)
}

func (s *Service) handle(path string, logging bool, handler interface{}, factory Factory) {
	route := fmt.Sprintf("%s.%s", s.Name, path)

	method := reflect.ValueOf(handler)
	s.checkHandler(method)

	reqT := method.Type().In(0)
	if reqT.Kind() == reflect.Ptr {
		reqT = reqT.Elem()
	}

	s.Transport.Handle(route, s.Name, func(data []byte) []byte {
		req := factory()
		req.SetError(s.Codec.Decode(data, req))

		s.logRequest(req, logging)

		res := method.Call([]reflect.Value{reflect.ValueOf(req)})[0].Interface()

		s.logResponse(req, res, logging)

		b, err := s.Codec.Encode(res)
		if err != nil {
			log.Fatal(err)
		}

		return b
	})
}

func (s *Service) handleHealthCheck(healthCheckName string, handler interface{}, factory Factory) {
	route := fmt.Sprintf("%s.%s", s.ID, healthCheckName)

	method := reflect.ValueOf(handler)
	s.checkHandler(method)

	reqT := method.Type().In(0)
	if reqT.Kind() == reflect.Ptr {
		reqT = reqT.Elem()
	}

	s.Transport.Handle(route, s.ID, func(data []byte) []byte {
		req := factory()
		req.SetError(s.Codec.Decode(data, req))

		s.logRequest(req, true)

		res := method.Call([]reflect.Value{reflect.ValueOf(req)})[0].Interface()

		s.logResponse(req, res, true)

		b, err := s.Codec.Encode(res)
		if err != nil {
			log.Fatal(err)
		}

		return b
	})
}

func (s *Service) RegisterHealthCheck(check *health.Dependency) {
	// We store the original check function
	realCheck := check.CheckIsWorking

	// And we change the original check function with another one which times out if
	// the check delays too much
	check.CheckIsWorking = func() (string, *oerror.Error) { return s.checkHealthOrTimeout(check.Name, check.Timeout, realCheck) }
	s.HealthChecks[check.Name] = *check

	// Then we restore the original object
	check.CheckIsWorking = realCheck
}

func (s *Service) checkHealthOrTimeout(name string, timeout time.Duration, check func() (string, *oerror.Error)) (string, *oerror.Error) {

	resultChannel := make(chan struct {
		str  string
		oerr *oerror.Error
	})
	timeoutChannel := make(chan bool)

	go func() {
		str, oerr := check()
		resultChannel <- struct {
			str  string
			oerr *oerror.Error
		}{
			str:  str,
			oerr: oerr,
		}
	}()

	go func() {
		time.Sleep(timeout)
		timeoutChannel <- true
	}()

	select {
	case res := <-resultChannel:
		return res.str, res.oerr
	case <-timeoutChannel:
		oerr := oerror.New(string(health.HC_CRIT))
		return "The health check " + name + " did timeout for " + string(timeout/time.Second) + " seconds", oerr
	}
}

// Call orion service
func (s *Service) Call(req interfaces.Request, raw interface{}) {
	res, ok := raw.(interfaces.Response)
	checkResponseCast(ok)

	closeTracer := s.Tracer.Trace(req)

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
			SetParams(err).
			SetLineOfCode(oerror.GenerateLOC(1)).
			Send()

		return
	}

	closeTracer()
}

func (s *Service) commsWithWatchdog() {
	var resultChannel chan interfaces.Response

	// This handles the loop for communications with the Watchdog
	s.closeWatchdogChannel, resultChannel = health.WatchdogRegisterLoop(s.Name, s.ID, s.Endpoints,
		func(endpoint string, request interfaces.Request) interfaces.Response {
			request.SetPath(s.WatchdogServiceName + endpoint)
			res := &response.Response{}
			s.Call(request, res)
			return res
		})

	// Wait at least for the first try
	<-resultChannel

	// For every other result, just print failures
	go func() {
		for s.closeWatchdogChannel != nil {
			res := <-resultChannel
			err := res.GetError()
			if err != nil {
				s.Logger.
					CreateMessage("Health/Watchdog").
					SetLevel(logger.ERROR).
					SetMap(map[string]interface{}{
						"description": "Error trying to register or ping the Watchdog '" + s.WatchdogServiceName + "' service.",
					}).
					Send()
			}
		}
	}()
}

func (s *Service) listenToHealthChecks() {
	// Status
	for name, check := range s.HealthChecks {
		s.handleHealthCheck("status."+name, health.DependencyHandleGenerator(check), health.DependencyFactory)
	}

	s.handleHealthCheck("status.am-i-up", health.AmIUpHandle, health.AmIUpFactory)
	s.handleHealthCheck("status.aggregate", health.AggregateHandleGenerator(s.HealthChecks), health.AggregateFactory)

	// s.Handle("status/about")
	// s.Handle("status/traverse")
}

// Listen to the transport protocol
func (s *Service) Listen(callback func()) {
	if s.EnableStatusEndpoints {
		s.listenToHealthChecks()
	}
	if s.RegisterToWatchdog {
		s.commsWithWatchdog()
	}
	s.Transport.Listen(callback)
}

func (s *Service) closeWatchdogLoop() {
	if s.closeWatchdogChannel != nil {
		s.closeWatchdogChannel <- true
		s.closeWatchdogChannel = nil
	}
}

// Close the transport protocol
func (s *Service) Close() {
	s.closeWatchdogLoop()
	s.Transport.Close()
}

// OnClose adds a handler to a transport connection closed event
func (s *Service) OnClose(handler func()) {
	s.Transport.OnClose(func(*nats.Conn) {
		s.closeWatchdogLoop()
		handler()
	})
}

// String return the name and the id of the service
func (s *Service) String() string {
	return fmt.Sprintf("%s-%s", s.Name, s.ID)
}

func (s *Service) logRequest(raw interface{}, logging bool) {
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

		s.Logger.
			CreateMessage(req.GetPath()).
			SetLevel(logger.INFO).
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
