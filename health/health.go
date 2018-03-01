package health

import (
	"time"

	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/betit/orion-go-sdk/interfaces"
	"github.com/betit/orion-go-sdk/request"
	"github.com/betit/orion-go-sdk/response"
)

const (
	// How long will take the service to ping the watchdog
	watchdogLoopPing = 1 * time.Minute

	// How long will take the service to retry the register to watchdog
	watchdogLoopRetry = 10 * time.Second

	// Timeout if watchdog is not answering
	watchdogTimeout = 10 * time.Second
)

func DefaultWatchdogServiceName() string {
	return "Watchdog"
}

type WatchdogPingRequest struct {
	request.Request
	ServiceID string `msgpack:"serviceId"`
	Name      string `msgpack:"name"`
}

type WatchdogRegisterRequest struct {
	request.Request
	ServiceID string   `msgpack:"serviceId"`
	Name      string   `msgpack:"name"`
	Endpoints []string `msgpack:"endpoints"`
}

func WatchdogRegisterLoop(
	name string,
	uid string,
	listOfEndpoints []string,
	requestToWatchdog func(endpoint string, request interfaces.Request) interfaces.Response) (closeChannel chan bool, responseChannel chan interfaces.Response) {
	closeChannel = make(chan bool)
	responseChannel = make(chan interfaces.Response)
	timeoutChannel := make(chan bool)
	loopTime := make(chan time.Duration, 1)

	endThisLoop := false
	imRegisteredInWatchdog := false

	// Firstly don't wait for register
	loopTime <- 0

	// Loop every loopTime time
	go func() {
		nextTime := <-loopTime
		time.Sleep(nextTime)
		if endThisLoop {
			return
		}
		timeoutChannel <- true
	}()

	pingRequest := WatchdogPingRequest{
		ServiceID:   uid,
		Name: name,
	}

	pingRequest.SetTimeoutDuration(watchdogTimeout)

	registerRequest := WatchdogRegisterRequest{
		ServiceID:        uid,
		Name:      name,
		Endpoints: listOfEndpoints,
	}

	// Then infinite loop of register/ping or close
	go func() {
		for {
			select {
			case <-timeoutChannel:
				if imRegisteredInWatchdog {
					// do ping
					res := requestToWatchdog("/ping", &pingRequest)
					everythingOk := res.GetError() == nil
					if everythingOk {
						loopTime <- watchdogLoopPing
					} else {
						imRegisteredInWatchdog = false

						err := res.GetError()
						if err.Code == WHO_ARE_YOU {
							// Try to register immediately: the watchdog doesn't know this service,
							// maybe because watchdog got restarted between two pings
							loopTime <- 0
						} else {
							loopTime <- watchdogLoopRetry
						}
					}
					responseChannel <- res
				} else {
					// do register
					res := requestToWatchdog("/register", &registerRequest)
					everythingOk := res.GetError() == nil
					if everythingOk {
						imRegisteredInWatchdog = true
						loopTime <- watchdogLoopPing
					} else {
						loopTime <- watchdogLoopRetry
					}
					responseChannel <- res
				}
			case <-closeChannel:
				endThisLoop = true
				responseChannel <- &response.Response{}
				return
			}
		}
	}()

	return closeChannel, responseChannel
}

type Dependency struct {
	CheckIsWorking func() (string, *oerror.Error)
	Timeout        time.Duration
	IsInternal     bool
	Name           string
}
