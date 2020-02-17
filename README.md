[![Build Status][travis-image]][travis-url] 
[![Test coverage][coveralls-image]][coveralls-url]
[![GoDoc][godoc-image]][godoc-url]

## Installation

```sh
$ go get github.com/gig/orion-go-sdk
```

or

```sh
$ dep ensure -add github.com/gig/orion-go-sdk
```

## Basic example

### Note - You will need to have running all of the orion [dependencies](https://github.com/gig/orion)

Add the following into `foo.go` and run `go run foo.go --verbose`

```go
package main

import (
	orion "github.com/gig/orion-go-sdk"
	"github.com/gig/orion-go-sdk/interfaces"
	"github.com/gig/orion-go-sdk/request"
	"github.com/gig/orion-go-sdk/response"
)

type params struct {
	A int `msgpack:"a"`
	B int `msgpack:"b"`
}

type addReq struct {
	request.Request
	Params params `msgpack:"params"`
}

type addPayload struct {
	Result int `msgpack:"result"`
}

type addRes struct {
	response.Response
	Payload addPayload `msgpack:"payload"`
}

func main() {
	svc := orion.New("calc")

	factory := func() interfaces.Request {
		return &addReq{}
	}

	handle := func(req *addReq) *addRes {
		return &addRes{
			Payload: addPayload{
				Result: req.Params.A + req.Params.B,
			},
		}
	}

	svc.Handle("add", handle, factory)

	svc.Listen(func() {
		svc.Logger.CreateMessage("ready").Send()
	})
}
```

Then add the following into `bar.go` and run `go run bar.go`

```go
package main

import (
	orion "github.com/gig/orion-go-sdk"
	"github.com/gig/orion-go-sdk/request"
	"github.com/gig/orion-go-sdk/response"
)

type params struct {
	A int `msgpack:"a"`
	B int `msgpack:"b"`
}

type addReq struct {
	request.Request
	Params params `msgpack:"params"`
}

type addPayload struct {
	Result int `msgpack:"result"`
}

type addRes struct {
	response.Response
	Payload addPayload `msgpack:"payload"`
}

func main() {
	svc := orion.New("calc")

	req := &addReq{
		Params: params{
			A: 1,
			B: 2,
		},
	}
	req.Path = "/calc/add"

	res := &addRes{}

	svc.Call(req, res)

	println(res.Payload.Result) // 3
}
```

You can find more examples in the test files.

## Health checks

Support for health checking is present if the services are running with the environment variable `WATCHDOG=true`. Also,
a [watchdog](https://github.com/GiG/orion-watchdog) must be running in the same environment as the service.

For instance, you could write an elasticsearch health check as:

```go
import (
    "context"
	"time"

	el "gopkg.in/olivere/elastic.v5"
	"github.com/gig/orion-go-sdk"
	"github.com/gig/orion-go-sdk/health"
)

func HealthCheck(client * el.Client) (*elastic.ClusterHealthResponse, error) {
	h := client.ClusterHealth()
	return h.Do(context.Background())
}

func ElasticHealthFactoryChecker(service * orion.Service, client * el.Client)  {
	service.RegisterHealthCheck(&health.Dependency{
		Name: "elasticsearch",
		Timeout: 30 * time.Second,
		CheckIsWorking: func () (string, *orion.Error) {
			res, err := HealthCheck(client)

			if err != nil {
				return "Health check went wrong: " + err.Error(), orion.ServiceError(string(health.HC_CRIT))
			}

			if res.Status == "yellow" {
				return "Non-green status: " + res.Status, orion.ServiceError(string(health.HC_WARN))
			}

			if res.Status == "red" {
				return "Non-green status: " + res.Status, orion.ServiceError(string(health.HC_CRIT))
			}

			return "", nil
		},
	})
}
```

## Tests

```bash
$ go test -v .
```

## License

[MIT](https://github.com/gig/orion-go-sdk/blob/master/LICENSE)

[travis-image]: https://travis-ci.org/GiG/orion-go-sdk.svg?branch=master
[travis-url]: https://travis-ci.org/GiG/orion-go-sdk/
[coveralls-image]: https://coveralls.io/repos/GiG/orion-go-sdk/badge.svg
[coveralls-url]: https://coveralls.io/r/GiG/orion-go-sdk
[godoc-image]: https://godoc.org/github.com/gig/orion-go-sdk?status.svg
[godoc-url]: https://godoc.org/github.com/gig/orion-go-sdk
