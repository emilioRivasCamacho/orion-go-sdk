package mock

import (
	"errors"
	"time"

	jsoncodec "github.com/gig/orion-go-sdk/codec/json"

	"github.com/gig/orion-go-sdk/interfaces"
)

type TransportMock struct {
	Subscriptions map[string]func([]byte)
	Handlers      map[string]func([]byte, func(interfaces.Response))
	Open          bool
	CloseHandler  func()
	codec         interfaces.Codec
}

var _ interfaces.Transport = (*TransportMock)(nil)

func New() *TransportMock {
	return &TransportMock{
		Handlers:      make(map[string]func([]byte, func(interfaces.Response))),
		Subscriptions: make(map[string]func([]byte)),
		codec:         jsoncodec.New(),
	}
}

func (t *TransportMock) Listen(f func()) {
	t.Open = true
	f()
}

func (t *TransportMock) Publish(s string, bytes []byte) error {
	if sub, ok := t.Subscriptions[s]; ok {
		sub(bytes)
		return nil
	} else {
		return errors.New("can not find path")
	}
}

func (t *TransportMock) Subscribe(s string, g string, f func([]byte)) error {
	t.Subscriptions["/"+g+"/"+s] = f
	return nil
}

func (t *TransportMock) SubscribeForRawMsg(s string, s2 string, f func(interface{})) error {
	panic("do not implement, this is unused and should be deprecated")
}

func (t *TransportMock) Handle(s string, g string, f func([]byte, func(response interfaces.Response))) error {
	t.Handlers["/"+g+"/"+s] = f
	return nil
}

func simulateTimeout(f func(), timeout time.Duration) bool {
	c := make(chan bool, 2)

	go func() {
		time.Sleep(timeout)
		c <- true
	}()

	go func() {
		f()
		c <- false
	}()

	return <-c
}

func (t *TransportMock) Request(s string, bytes []byte, timeout int) ([]byte, error) {
	if sub, ok := t.Handlers[s]; ok {
		resp := make(chan interfaces.Response, 1)
		timeout := simulateTimeout(func() {
			sub(bytes, func(response interfaces.Response) {
				resp <- response
			})
		}, time.Duration(timeout)*time.Millisecond)

		if timeout {
			return nil, errors.New("timed out")
		}

		encoded, _ := t.codec.Encode(<-resp)
		return encoded, nil
	} else {
		return nil, errors.New("can not find path")
	}
}

func (t *TransportMock) Close() {
	t.Open = false
	if t.CloseHandler != nil {
		t.CloseHandler()
	}
}

func (t *TransportMock) IsOpen() bool {
	return t.Open
}

func (t *TransportMock) OnClose(i interface{}) {
	t.CloseHandler = i.(func())
}

type RegisterMock struct {
	Callback func(serviceName string, instanceName string, prefixList []string) error
}

func (r *RegisterMock) Register(serviceName string, instanceName string, prefixList []string) error {
	return r.Callback(serviceName, instanceName, prefixList)
}

var _ interfaces.Register = (*RegisterMock)(nil)
