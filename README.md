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
	"log"

	orion "github.com/betit/orion-go-sdk"
)

func main() {
	svc := orion.New("foo")

	svc.Handle("get", func(req *orion.Request) *orion.Response {
		res := &orion.Response{}
		err := res.SetPayload("bar")
		if err != nil {
			log.Fatal(err)
		}

		return res
	})

	svc.Listen(func() {
		svc.Logger.CreateMessage("ready").Send()
	})
}
```

Then add the following into `bar.go` and run `go run bar.go`

```go
package main

import (
	"log"

	orion "github.com/betit/orion-go-sdk"
)

func main() {
  svc := orion.New("bar")

	req := &orion.Request{
		Path: "/foo/get",
	}

	res := svc.Call(req)
	if res.Error != nil {
		log.Fatal(res.Error.Message)
	}

	payload := ""

	err := res.ParsePayload(&payload)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Print(payload)
}
```

You can find more detailed examples in the `examples` folder.

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
