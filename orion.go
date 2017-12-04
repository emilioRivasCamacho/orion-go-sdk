package orion

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/betit/orion-go-sdk/codec/msgpack"
	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/betit/orion-go-sdk/interfaces"
	"github.com/betit/orion-go-sdk/logger"
	"github.com/betit/orion-go-sdk/tracer"
	"github.com/betit/orion-go-sdk/transport/nats"
	uuid "github.com/satori/go.uuid"
)

// Factory func type - the one that creates the req obj
type Factory = func() interfaces.Request

// Service for orion
type Service struct {
	ID        string
	Name      string
	Timeout   int
	Codec     interfaces.Codec
	Transport interfaces.Transport
	Tracer    interfaces.Tracer
	Logger    interfaces.Logger
}

// New orion service
func New(name string, options ...Option) *Service {
	opts := &Options{}

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

	return &Service{
		ID:        uuid.NewV4().String(),
		Name:      name,
		Timeout:   200,
		Codec:     opts.Codec,
		Transport: opts.Transport,
		Tracer:    opts.Tracer,
		Logger:    opts.Logger,
	}
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

// Call orion service
func (s *Service) Call(req interfaces.Request, raw interface{}) {
	res, ok := raw.(interfaces.Response)
	checkResponseCast(ok)

	closeTracer := s.Tracer.Trace(req)

	encoded, err := s.Codec.Encode(req)
	if err != nil {
		res.SetError(oerror.New("ORION_ENCODE").SetMessage(err.Error()))
		return
	}

	path := replaceOmitEmpty(req.GetPath(), "/", ".")
	b, err := s.Transport.Request(path, encoded, s.getTimeout(req))
	if err != nil {
		res.SetError(oerror.New("ORION_TRANSPORT").SetMessage(err.Error()))
		return
	}

	err = s.Codec.Decode(b, res)
	if err != nil {
		res.SetError(oerror.New("ORION_DECODE").SetMessage(err.Error()))
		return
	}

	closeTracer()
}

// Listen to the transport protocol
func (s *Service) Listen(callback func()) {
	s.Transport.Listen(callback)
}

// Close the transport protocol
func (s *Service) Close() {
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
