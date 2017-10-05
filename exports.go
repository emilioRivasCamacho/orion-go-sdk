package orion

import (
	"github.com/betit/orion/go/error"
	"github.com/betit/orion/go/request"
	"github.com/betit/orion/go/response"
)

// Request from microservice
type Request = orequest.Request

// Response from microservice
type Response = oresponse.Response

// Error for orion
type Error = oerror.Error

// ServiceError for orion
var ServiceError = oerror.New
