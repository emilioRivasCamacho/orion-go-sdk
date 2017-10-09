package orion

import (
	"fmt"
	"log"
	"strings"

	"github.com/betit/orion-go-sdk/codec/msgpack"
	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/betit/orion-go-sdk/interfaces"
	"github.com/betit/orion-go-sdk/logger"
	"github.com/betit/orion-go-sdk/tracer"
	"github.com/betit/orion-go-sdk/transport/nats"
	uuid "github.com/satori/go.uuid"
)

// Service for orion
type Service struct {
	ID          string
	Name        string
	CallTimeOut int
	codec       interfaces.Codec
	transport   interfaces.Transport
	tracer      interfaces.Tracer
	Logger      interfaces.Logger
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

	if opts.Codec == nil {
		opts.Codec = msgpack.New()
	}

	if opts.Tracer == nil {
		opts.Tracer = tracer.New(name)
	}

	if opts.Logger == nil {
		opts.Logger = logger.New(name)
	}

	return &Service{
		ID:          uuid.NewV4().String(),
		Name:        name,
		CallTimeOut: 200,
		codec:       opts.Codec,
		transport:   opts.Transport,
		tracer:      opts.Tracer,
		Logger:      opts.Logger,
	}
}

// Emit to services
func (s *Service) Emit(topic string, data interface{}) error {
	msg, err := s.codec.Encode(data)
	if err != nil {
		return err
	}
	return s.transport.Publish(topic, msg)
}

// On service emit
func (s *Service) On(topic string, handler func([]byte)) {
	subject := fmt.Sprintf("%s:%s", s.Name, topic)
	s.transport.Subscribe(subject, s.Name, handler)
}

// Decode bytes to passed interface
func (s *Service) Decode(data []byte, to interface{}) error {
	return s.codec.Decode(data, &to)
}

// HandleWithoutLogging enabled
func (s *Service) HandleWithoutLogging(path string, handler func(*Request) *Response) {
	logging := false
	s.handle(path, logging, handler)
}

// Handle has enabled logging. What that means is when the request
// arrives the service will log the request including the raw params. Once the
// response is returned, the service will check for error and if there is such,
// the error will be logged
func (s *Service) Handle(path string, handler func(*Request) *Response) {
	logging := true
	s.handle(path, logging, handler)
}

func (s *Service) handle(path string, logging bool, handler func(*Request) *Response) {
	route := fmt.Sprintf("%s.%s", s.Name, path)
	s.transport.Handle(route, s.Name, func(data []byte) []byte {
		var req Request
		err := s.codec.Decode(data, &req)
		if err != nil {
			log.Fatal(err)
		}

		params := map[string]interface{}{
			"raw":  string(req.Params),
			"meta": req.Meta,
		}

		if logging {
			s.Logger.
				CreateMessage(path).
				SetLevel(logger.INFO).
				SetID(req.GetID()).
				SetMap(params).
				Send()
		}

		res := handler(&req)

		if res.Error != nil && logging {
			s.Logger.
				CreateMessage(path).
				SetLevel(logger.ERROR).
				SetID(req.GetID()).
				SetParams(res.Error).
				Send()
		}

		dat, err := s.codec.Encode(res)
		if err != nil {
			log.Fatal(err)
		}

		return dat
	})
}

// Call orion service
func (s *Service) Call(request *Request) *Response {
	closeTracer := s.tracer.Trace(request)
	response := Response{}

	encoded, err := s.codec.Encode(request)
	if err != nil {
		response.Error = oerror.New("ORION_ENCODE")
		response.Error.SetMessage(err.Error())
		return &response
	}

	path := replaceOmitEmpty(request.Path, "/", ".")
	res, err := s.transport.Request(path, encoded, s.getTimeout(request))
	if err != nil {
		response.Error = oerror.New("ORION_TRANSPORT")
		response.Error.SetMessage(err.Error())
		return &response
	}
	err = s.codec.Decode(res, &response)
	if err != nil {
		response.Error = oerror.New("ORION_DECODE")
		response.Error.SetMessage(err.Error())
		return &response
	}
	closeTracer()
	return &response
}

// Listen to the transport protocol
func (s *Service) Listen(callback func()) {
	s.transport.Listen(callback)
}

// Close the transport protocol
func (s *Service) Close() {
	s.transport.Close()
}

// String return the name and the id of the service
func (s *Service) String() string {
	return fmt.Sprintf("%s-%s", s.Name, s.ID)
}

func (s *Service) getTimeout(request *Request) int {
	if request.CallTimeout != nil {
		return *request.CallTimeout
	}
	return s.CallTimeOut
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
