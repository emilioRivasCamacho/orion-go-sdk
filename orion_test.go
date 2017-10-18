package orion

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var svc = New("e2e")

func TestHandle(t *testing.T) {
	type params struct {
		A int
		B int
	}
	expected := 3
	done := make(chan int)

	calc := New("calc")

	calc.Handle("sum", func(req *Request) *Response {
		res := &Response{}

		p := &params{}
		req.GetParams(p)

		res.SetPayload(p.A + p.B)
		return res
	})

	go calc.Listen(func() {
		var result int

		req := &Request{}
		req.Path = "/calc/sum"
		req.SetParams(params{
			A: 1,
			B: 2,
		})

		res := svc.Call(req)
		res.ParsePayload(&result)

		calc.Close()

		done <- result
	})

	result := <-done
	assert.Equal(t, expected, result)
}

func TestTimeout(t *testing.T) {
	done := make(chan bool)

	s := New("timeout")

	s.Handle("test", func(req *Request) *Response {
		time.Sleep(201 * time.Millisecond)
		return &Response{}
	})

	go s.Listen(func() {
		req := &Request{}
		req.Path = "/timeout/test"

		res := svc.Call(req)
		if res.Error == nil {
			// the call must timeout, the default timeout is 200ms
			done <- true
			return
		}

		req = &Request{}
		req.Path = "/timeout/test"
		timeout := 300
		req.Timeout = &timeout

		res = svc.Call(req)
		s.Close()

		done <- res.Error != nil
	})

	hasError := <-done
	assert.Equal(t, false, hasError)
}

func TestPubSub(t *testing.T) {
	done := make(chan bool)

	pubsub := New("pubsub")

	pubsub.On("event", func(args []byte) {
		done <- true
	})

	go pubsub.Listen(func() {
		svc.Emit("pubsub:event", nil)
	})

	success := <-done
	assert.Equal(t, true, success)
}

func TestMain(m *testing.M) {
	svc.Listen(func() {
		tests := m.Run()
		os.Exit(tests)
	})
}
