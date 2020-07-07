package orion

import (
	"testing"
	"time"

	"github.com/gig/orion-go-sdk/interfaces"
	"github.com/stretchr/testify/assert"
)

var svc *Service

func DisableHealthChecks(opt *Options) {
	opt.DisableHealthChecks = true
}

func TestHandle(t *testing.T) {
	type params struct {
		A int
		B int
	}
	expected := 3
	done := make(chan int)

	calc := New("calc", DisableHealthChecks)

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

		calc.Close()

		done <- result
	})

	result := <-done
	assert.Equal(t, expected, result)
}

func TestTimeout(t *testing.T) {
	done := make(chan bool)

	s := New("timeout", DisableHealthChecks)

	handler := func(req *Request) *Response {
		time.Sleep(201 * time.Millisecond)
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
	done := make(chan bool)

	pubsub := New("pubsub", DisableHealthChecks)

	pubsub.On("event", func(args []byte) {
		done <- true
	})

	go pubsub.Listen(func() {
		svc.Emit("pubsub:event", nil)
	})

	success := <-done
	assert.Equal(t, true, success)
}

func TestOnClose(t *testing.T) {
	done := make(chan bool)

	onclose := New("onclose", DisableHealthChecks)

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

	calc := New("calc", DisableHealthChecks)

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

func TestPathCalculation(t *testing.T) {
	svc = New("e2e", DisableHealthChecks)
	route := svc.getRouteFromPath("action")
	assert.Equal(t, "e2e.action", route)

	route = svc.getRouteFromPath("module/action")
	assert.Equal(t, "module.action", route)
}

func TestCustomModuleName(t *testing.T) {
	type params struct {
		A int
		B int
	}
	expected := 3
	done := make(chan int)

	calc := New("calc", DisableHealthChecks)

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

		res.ParsePayload(&result)

		calc.Close()

		done <- result
	})

	result := <-done
	assert.Equal(t, expected, result)
}

/*func TestMain(m *testing.M) {
	svc = New("e2e", DisableHealthChecks)
	svc.Listen(func() {
		tests := m.Run()
		os.Exit(tests)
	})
}*/
