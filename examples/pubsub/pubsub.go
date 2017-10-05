package main

import (
	"fmt"
	"time"

	orion "github.com/betit/orion-go-sdk"
)

var (
	svc        = orion.New("examples")
	dateLayout = "Mon Jan 02 2006 15:04:05 GMT-0700 (MST)"
)

type nested struct {
	Time string `json:"time" msgpack:"time"`
}

type expected struct {
	Nexted nested `json:"nested" msgpack:"nested"`
}

func main() {
	svc.On("go", func(res []byte) {
		payload := &expected{}
		svc.Decode(res, payload)
		fmt.Printf("%+v\n", payload)
	})
	svc.Listen(func() {
		payload := &expected{}
		payload.Nexted.Time = time.Now().Format(dateLayout)
		svc.Emit("examples:go", payload)
		svc.Emit("examples:node", payload)
	})
}
