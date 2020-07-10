package orion

import (
	"errors"
	"testing"
	"time"

	"github.com/gig/orion-go-sdk/health"
	"github.com/gig/orion-go-sdk/transport/mock"

	"github.com/gig/orion-go-sdk/interfaces"
	"github.com/stretchr/testify/assert"
)

func DisableHealthChecks(opt *Options) {
	opt.DisableHealthChecks = true
}

var svc *Service

func prepareServiceCaller(transport *mock.TransportMock) *Service {
	return New("caller", DisableHealthChecks, SetTransport(transport))
}

func TestHandleNoHealthCheck(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	type params struct {
		A int
		B int
	}
	expected := 3
	done := make(chan int)

	calc := New("calc", DisableHealthChecks, SetTransport(transport))

	factory := func() interfaces.Request {
		return &Request{}
	}

	handle := func(req *Request) *Response {
		res := &Response{}

		p := &params{}
		req.ParseParams(p)

		res.SetPayload(p.A + p.B)
		return res
	}

	calc.Handle("sum", handle, factory)

	go calc.Listen(func() {
		var result int

		req := &Request{}
		req.SetPath("/calc/sum").SetParams(params{
			A: 1,
			B: 2,
		})

		res := &Response{}
		svc.Call(req, res)

		res.ParsePayload(&result)
		if res.Error != nil {
			assert.NoError(t, res.Error)
		}

		calc.Close()

		done <- result
	})

	result := <-done
	assert.Equal(t, expected, result)
}

func TestTimeout(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	done := make(chan bool)

	s := New("timeout", SetTransport(transport))

	handler := func(req *Request) *Response {
		time.Sleep(250 * time.Millisecond)
		return &Response{}
	}

	factory := func() interfaces.Request {
		return &Request{}
	}

	s.Handle("test", handler, factory)

	go s.Listen(func() {
		req := &Request{}
		req.Path = "/timeout/test"

		res1 := &Response{}
		svc.Call(req, res1)
		if res1.GetError() == nil {
			// the call must timeout, the default timeout is 200ms
			done <- true
			return
		}

		req = &Request{}
		req.SetPath("/timeout/test").SetTimeout(300)

		res2 := &Response{}
		svc.Call(req, res2)
		s.Close()

		done <- res2.GetError() != nil
	})

	hasError := <-done
	assert.Equal(t, false, hasError)
}

func TestPubSub(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	done := make(chan bool)

	pubsub := New("pubsub", SetTransport(transport))

	pubsub.On("event", func(args []byte) {
		done <- true
	})

	go pubsub.Listen(func() {
		err := svc.Emit("pubsub:event", nil)
		assert.NoError(t, err)
	})

	success := <-done
	assert.Equal(t, true, success)
}

func TestOnClose(t *testing.T) {
	transport := mock.New()

	done := make(chan bool)

	onclose := New("onclose", SetTransport(transport))

	onclose.OnClose(func() {
		done <- true
	})

	go onclose.Listen(func() {
		onclose.Close()
	})

	success := <-done
	assert.Equal(t, true, success)
}

func TestCustomReqRes(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	type params struct {
		C int
		D int
	}

	type customReq struct {
		Request
		Params params
	}

	type customPayload struct {
		Result int
	}

	type customRes struct {
		Response
		Payload customPayload
	}

	expected := 3
	done := make(chan int)

	calc := New("calc", SetTransport(transport))

	factory := func() interfaces.Request {
		return &customReq{}
	}

	handle := func(req *customReq) *customRes {
		return &customRes{
			Payload: customPayload{
				Result: req.Params.C + req.Params.D,
			},
		}
	}

	calc.Handle("sum", handle, factory)

	go calc.Listen(func() {
		res := &customRes{}
		req := &customReq{
			Params: params{
				C: 1,
				D: 2,
			},
		}
		req.SetPath("/calc/sum")

		svc.Call(req, res)

		calc.Close()

		done <- res.Payload.Result
	})

	result := <-done
	assert.Equal(t, expected, result)
}

func TestCustomModuleName(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	type params struct {
		A int
		B int
	}
	expected := 3
	done := make(chan int)

	calc := New("calc", SetTransport(transport))

	factory := func() interfaces.Request {
		return &Request{}
	}

	handle := func(req *Request) *Response {
		res := &Response{}

		p := &params{}
		req.ParseParams(p)

		res.SetPayload(p.A + p.B)
		return res
	}

	calc.Handle("math/sum", handle, factory)

	go calc.Listen(func() {
		var result int

		req := &Request{}
		req.SetPath("/math/sum").SetParams(params{
			A: 1,
			B: 2,
		})

		res := &Response{}
		svc.Call(req, res)

		if res.Error != nil {
			assert.NoError(t, res.Error)
		}

		res.ParsePayload(&result)

		calc.Close()

		done <- result
	})

	result := <-done
	assert.Equal(t, expected, result)
}

func TestOrionDecodeError(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	type response struct {
		Response
		Params struct {
			A int
			B int
		} `json:"params"`
	}

	type wrongResponse struct {
		Response
		Params struct {
			A bool
			B string
		} `json:"params"`
	}

	done := make(chan *Error)

	test := New("test", SetTransport(transport))

	factory := func() interfaces.Request {
		return &Request{}
	}

	handle := func(req *Request) *response {
		return &response{Params: struct {
			A int
			B int
		}{A: 14, B: 15}}
	}

	test.Handle("/endpoint", handle, factory)

	go test.Listen(func() {
		req := &Request{}
		req.SetPath("/test/endpoint")

		res := &wrongResponse{}
		svc.Call(req, res)

		done <- res.Error

		test.Close()
	})

	result := <-done
	assert.Equal(t, "ORION_DECODE", result.Code)
	assert.Error(t, result)
}

func TestDisableHealthCheck(t *testing.T) {
	transport := mock.New()
	health := New("withhealth", SetTransport(transport))
	noHealth := New("nohealth", DisableHealthChecks, SetTransport(transport))

	wait := make(chan struct{})

	go health.Listen(func() { wait <- struct{}{} })
	go noHealth.Listen(func() { wait <- struct{}{} })

	<-wait
	<-wait

	assert.Contains(t, transport.Handlers, "/"+health.InstanceName+"/healthcheck")
	assert.NotContains(t, transport.Handlers, "/"+noHealth.InstanceName+"/healthcheck")
}

func TestHealthCheckGood(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	goodHealth := New("healthy", SetTransport(transport))

	goodHealth.RegisterHealthCheck(&health.Dependency{
		CheckIsWorking: func() (health.HealthCheckResult, error) {
			return health.HC_OK, nil
		},
		Timeout: 100,
		Name:    "healthy",
	})

	wait := make(chan struct{})

	go goodHealth.Listen(func() { wait <- struct{}{} })

	<-wait

	res := &health.HealthCheckResponse{}

	svc.Call(&Request{
		Path: "/" + goodHealth.InstanceName + "/healthcheck",
	}, res)

	if res.Error != nil {
		assert.NoError(t, res.Error)
	}
}

func TestHealthCheckBad(t *testing.T) {
	transport := mock.New()
	svc := prepareServiceCaller(transport)
	defer svc.Close()

	badHealth := New("unhealthy", SetTransport(transport))

	badHealth.RegisterHealthCheck(&health.Dependency{
		CheckIsWorking: func() (health.HealthCheckResult, error) {
			return health.HC_CRIT, errors.New("unhealthy error")
		},
		Timeout: 100,
		Name:    "unhealthy",
	})

	wait := make(chan struct{})

	go badHealth.Listen(func() { wait <- struct{}{} })

	<-wait

	res := &health.HealthCheckResponse{}

	svc.Call(&Request{
		Path: "/" + badHealth.InstanceName + "/healthcheck",
	}, res)

	assert.Error(t, res.Error)
}

func TestRegisterNames(t *testing.T) {
	nameToAssert := "testing_register"
	instanceNames := make(map[string]struct{})

	opts := func(options *Options) {
		options.Transport = mock.New()
		options.Register = &mock.RegisterMock{Callback: func(serviceName string, instanceName string, prefixList []string) error {
			assert.Equal(t, nameToAssert, serviceName, "service name must be the same")
			assert.NotContains(t, instanceNames, instanceName, "instance names must be unique")
			instanceNames[instanceName] = struct{}{}
			return nil
		}}
	}

	svc1 := New(nameToAssert, opts)
	svc2 := New(nameToAssert, opts)
	svc3 := New(nameToAssert, opts)

	wait := make(chan struct{})

	go svc1.Listen(func() { wait <- struct{}{} })
	<-wait

	go svc2.Listen(func() { wait <- struct{}{} })
	<-wait

	go svc3.Listen(func() { wait <- struct{}{} })
	<-wait

}
