package main

import (
	"fmt"
	"time"

	orion "github.com/betit/orion-go-sdk"
	logger "github.com/betit/orion-go-sdk/logger"
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
		svc.Logger.CreateMessage("TEST").SetLevel(logger.CRITICAL).Send()
		svc.Logger.CreateMessage("TEST").SetLevel(logger.EMERGENCY).Send()
		svc.Logger.CreateMessage("TEST").SetLevel(logger.ERROR).Send()
		svc.Logger.CreateMessage("TEST").SetLevel(logger.ALERT).Send()
		svc.Logger.CreateMessage("TEST").SetLevel(logger.WARNING).Send()
		svc.Logger.CreateMessage("TEST").SetLevel(logger.INFO).Send()
		svc.Logger.CreateMessage("TEST").SetLevel(logger.NOTICE).Send()
		svc.Logger.CreateMessage("TEST").SetLevel(logger.DEBUG).Send()
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
	req.GetParams(&params)
	from := req.Meta["time"]
	metaTime, _ := time.Parse(dateLayout, from)
	fmt.Printf("parsed meta time %+v\n", metaTime)
	fmt.Printf("req params %+v \n", params)
	fmt.Printf("req id %+v \n", req.GetID())

	res := orion.Response{}

	svc.Logger.CreateMessage("dummy").SetLevel(logger.INFO).SetID(req.GetID()).SetParams([]byte(from)).Send()

	m := map[string]interface{}{
		"test": "params",
		"what": "wtf",
	}
	svc.Logger.CreateMessage("dummy").SetLevel(logger.INFO).SetID(req.GetID()).SetMap(m).Send()

	res.Error = orion.ServiceError("1021312").SetMessage("some error message")
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
		Path:    "/examples-go/dummy",
		Timeout: &timeout,
	}
	c := complexStruct{
		Bar: simpleStruct{
			Foo: "bar",
			N:   100123,
			Now: time.Now().Format(dateLayout),
		},
	}
	req.SetParams(c)

	res := svc.Call(&req)

	payload := &complexStruct{}
	res.ParsePayload(&payload)
}

func traceRequests() {
	req := orion.Request{}
	req.Path = "/examples-go/level1"
	res := svc.Call(&req)
	p := complexStruct{}
	res.ParsePayload(&p)
	fmt.Printf("traceRequests response %+v\n", p)
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
	return svc.Call(req)
}
