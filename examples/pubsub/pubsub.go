package main

import (
	"fmt"

	"github.com/gig/orion-go-sdk"
	"github.com/gig/orion-go-sdk/interfaces"
)

type TestRequest struct {
	orion.Request
	Params map[string]interface{} `json:"params"`
}

type TestResponse struct {
	orion.Response
	Payload map[string]string `json:"payload"`
}

func main() {
	svc := orion.New("gopubsub")

	svc.Handle("test", func(req *TestRequest) interfaces.Response {
		fmt.Println(*req)
		return &TestResponse{
			Payload: map[string]string{
				"key": "value",
			},
		}
	}, func() interfaces.Request {
		return &TestRequest{}
	})

	svc.On("event", func(bytes []byte) {
		fmt.Println(string(bytes))
	})

	svc.Listen(func() {
		fmt.Println("listening")
	})
}
