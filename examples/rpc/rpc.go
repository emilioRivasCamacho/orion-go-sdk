package main

import (
	"encoding/json"
	"fmt"
	"time"

	orion "github.com/betit/orion-go-sdk"
)

var (
	svc        = orion.New("examples-go")
	dateLayout = "Mon Jan 02 2006 15:04:05 GMT-0700 (MST)"
)

type simpleStruct struct {
	Foo string `json:"foo" msgpack:"foo"`
	N   int64  `json:"n" msgpack:"n"`
	Now string `json:"now" msgpack:"now"`
}

type complexStruct struct {
	Bar simpleStruct `json:"bar" msgpack:"bar"`
}

func main() {
	runService(func() {
		runClient()
		traceRequests()
	})
}

func runService(callback func()) {
	svc.Handle("dummy", dummyHandle)
	svc.Handle("level1", level1Handle)
	svc.Handle("level2", level2Handle)
	svc.Handle("level3", level3Handle)
	svc.Handle("level4", level4Handle)

	svc.Listen(callback)
}

func dummyHandle(req *orion.Request) *orion.Response {
	params := complexStruct{}
	json.Unmarshal(req.Params, &params)
	fmt.Printf("req meta %+v \n", req.Meta)
	from := req.Meta["time"]
	metaTime, _ := time.Parse(dateLayout, from)
	fmt.Printf("parsed meta time %+v\n", metaTime)
	fmt.Printf("req params %+v \n", params)
	fmt.Printf("req id %+v \n", req.GetID())

	res := orion.Response{}

	svc.Logger.CreateMessage("dummy").SetLevel(6).SetID(req.GetID()).SetParams([]byte(from)).Send()

	m := map[string]interface{}{
		"test": "params",
		"what": "wtf",
	}
	svc.Logger.CreateMessage("dummy").SetLevel(6).SetID(req.GetID()).SetMap(m).Send()

	res.Error = orion.ServiceError("1021312")
	res.Error.SetMessage("some error message")
	res.SetPayload(params)
	return &res
}

func runClient() {
	timeout := 1000
	req := orion.Request{
		Meta: map[string]string{
			"sid":  "1234567",
			"time": time.Now().Format(dateLayout),
		},
		Path:        "/examples-go/dummy",
		CallTimeout: &timeout,
	}
	c := complexStruct{
		Bar: simpleStruct{
			Foo: "bar",
			N:   100123,
			Now: time.Now().Format(dateLayout),
		},
	}
	req.SetParams(&c)

	res := svc.Call(&req)

	// first approach
	payload := &complexStruct{}
	res.GetPayload(&payload)
	fmt.Printf("response approach 1 %+v\n", payload)

	// second approach
	result := &complexStruct{}
	json.Unmarshal(res.Payload, result)
	fmt.Printf("response approach 2 %+v\n", result)
}

func traceRequests() {
	req := orion.Request{}
	req.Path = "/examples-go/level1"
	svc.Call(&req)
	fmt.Printf("req id %+v \n", req.GetID())
}

func level1Handle(req *orion.Request) *orion.Response {
	req.Path = "/examples-go/level2"
	return svc.Call(req)
}

func level2Handle(req *orion.Request) *orion.Response {
	req.Path = "/examples-go/level3"
	return svc.Call(req)
}

func level3Handle(req *orion.Request) *orion.Response {
	req.Path = "/examples-go/level4"
	return svc.Call(req)
}

func level4Handle(req *orion.Request) *orion.Response {
	req.Path = "/examples-node/dummy"
	svc.Call(req)
	return &orion.Response{}
}
