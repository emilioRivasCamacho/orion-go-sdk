[![Build Status][travis-image]][travis-url] 
[![Test coverage][coveralls-image]][coveralls-url]
[![GoDoc][godoc-image]][godoc-url]

## Installation

```sh
$ go get github.com/betit/orion-go-sdk
```

or

```sh
$ dep ensure -add github.com/betit/orion-go-sdk
```

## Basic example

### Note - You will need to have running all of the orion [dependencies](https://github.com/betit/orion/tree/dev#development)

Add the following into `foo.go` and run `go run foo.go --verbose`

```go
package main

import (
	orion "github.com/betit/orion-go-sdk"
	"github.com/betit/orion-go-sdk/interfaces"
	"github.com/betit/orion-go-sdk/request"
	"github.com/betit/orion-go-sdk/response"
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
	orion "github.com/betit/orion-go-sdk"
	"github.com/betit/orion-go-sdk/request"
	"github.com/betit/orion-go-sdk/response"
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

## Tests

```bash
$ go test -v .
```

## License

[MIT](https://github.com/betit/orion-go-sdk/blob/master/LICENSE)

[travis-image]: https://travis-ci.org/betit/orion-go-sdk.svg?branch=master
[travis-url]: https://travis-ci.org/betit/orion-go-sdk/
[coveralls-image]: https://coveralls.io/repos/betit/orion-go-sdk/badge.svg
[coveralls-url]: https://coveralls.io/r/betit/orion-go-sdk
[godoc-image]: https://godoc.org/github.com/betit/orion-go-sdk?status.svg
[godoc-url]: https://godoc.org/github.com/betit/orion-go-sdk
