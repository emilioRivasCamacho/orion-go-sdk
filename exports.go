package orion

import (
	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/betit/orion-go-sdk/request"
	"github.com/betit/orion-go-sdk/response"
)

// Request from microservice
type Request = request.Request

// Response from microservice
type Response = response.Response

// Error for orion
type Error = oerror.Error

// ServiceError for orion
var ServiceError = oerror.New
