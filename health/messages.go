package health

import (
	oerror "github.com/betit/orion-go-sdk/error"
	"github.com/betit/orion-go-sdk/interfaces"
	"github.com/betit/orion-go-sdk/request"
	"github.com/betit/orion-go-sdk/response"
)

const (
	OK          = "OK"
	WHO_ARE_YOU = "WHO_ARE_YOU"
)

type AggregationType string

const (
	INTERNAL AggregationType = "internal"
	EXTERNAL AggregationType = "external"
)

type HealthCheckResult string

const (
	HC_OK   HealthCheckResult = "OK"
	HC_WARN HealthCheckResult = "WARN"
	HC_CRIT HealthCheckResult = "CRIT"
)

func AmIUpHandle(req *AmIUpRequest) *AmIUpResponse {
	return &AmIUpResponse{
		Payload: struct {
			Status string `msgpack:"status"`
		}{Status: OK},
	}
}

type AmIUpResponse struct {
	response.Response
	Payload struct {
		Status string `msgpack:"status"`
	} `msgpack:"params"`
}

type AmIUpRequest struct {
	request.Request
}

func AmIUpFactory() interfaces.Request {
	return &AmIUpRequest{}
}

type AggregateParams struct {
	Type *AggregationType `msgpack:"type"`
}

type AggregateRequest struct {
	request.Request
	Params AggregateParams `msgpack:"params"`
}

type DependencyCheckResult struct {
	Description *string           `msgpack:"description"`
	Result      HealthCheckResult `msgpack:"result"`
	Details     *string           `msgpack:"details"`
}

func checkToStructure(checkName string, str string, err *oerror.Error) DependencyCheckResult {
	res := DependencyCheckResult{}
	if err != nil {
		res.Result = HealthCheckResult(err.Code)
		res.Details = &str
		description := "Health check for " + checkName
		res.Description = &description
	} else {
		res.Result = HC_OK
	}
	return res
}

type AggregateResponse struct {
	response.Response
	Payload []DependencyCheckResult `msgpack:"payload"`
}

func AggregateHandleGenerator(checks map[string]Dependency) func(req *AggregateRequest) *AggregateResponse {
	return func(req *AggregateRequest) *AggregateResponse {
		res := &AggregateResponse{}
		if req.Error != nil {
			err := req.Error
			res.Error = oerror.New("REQUEST_ERROR")
			res.Error.SetMessage(err.Error())
		} else {
			for name, check := range checks {
				if req.Params.Type == nil || (*req.Params.Type == INTERNAL) == check.IsInternal {
					str, err := check.CheckIsWorking()
					res.Payload = append(res.Payload, checkToStructure(name, str, err))
				}
			}
		}
		return res
	}
}

func AggregateFactory() interfaces.Request {
	return &AggregateRequest{}
}

type DependencyParams struct {
	Type AggregationType `msgpack:"type"`
}

type DependencyRequest struct {
	request.Request
}

type DependencyResponse struct {
	response.Response
	Payload DependencyCheckResult `msgpack:"payload"`
}

func DependencyHandleGenerator(check Dependency) func(req *DependencyRequest) *DependencyResponse {
	return func(req *DependencyRequest) *DependencyResponse {
		res := &DependencyResponse{}
		if req.Error != nil {
			err := req.Error
			res.Error = oerror.New("REQUEST_ERROR")
			res.Error.SetMessage(err.Error())
		} else {
			str, err := check.CheckIsWorking()
			res.Payload = checkToStructure(check.Name, str, err)
		}
		return res
	}
}

func DependencyFactory() interfaces.Request {
	return &DependencyRequest{}
}
