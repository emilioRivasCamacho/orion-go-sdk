package tracer

import (
	"fmt"
	"github.com/opentracing/opentracing-go/ext"
	"io"
	"os"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"

	"github.com/gig/orion-go-sdk/env"
	"github.com/gig/orion-go-sdk/interfaces"
)

var (
	// to enable tracing, set this to true using TRACER_ENABLED env. var.
	enabled = false
	// SameSpan can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	SameSpan = true

	// TraceID128Bit generate 128 bit traceID's for root spans.
	TraceID128Bit = true

	// Debug tracer
	Debug = false

	// HostAddr of the tracer
	HostAddr = ""

	// Endpoint of the collector
	Endpoint = ""

)

// Tracer for orion
type Tracer struct {
	tracer opentracing.Tracer
	closer io.Closer
}


func (t Tracer) Close() error {
	return t.closer.Close()
}

var _ io.Closer = (*Tracer)(nil)

func init() {
	setVariables()
}

// New Jaeger tracer
func New(service string) *Tracer {
	cfg, err := config.FromEnv()

	if err != nil {
		fmt.Printf("cannot parse Jaeger env vars: %+v\n", err)
		os.Exit(1)
	}

	cfg.ServiceName = service
	cfg.Sampler.Type = "const"
	cfg.Sampler.Param = 1

	tracer, closer, err := cfg.NewTracer(config.Gen128Bit(true))

	if err != nil {
		fmt.Printf("unable to create tracer: %+v\n", err)
		os.Exit(1)
	}

	opentracing.SetGlobalTracer(tracer)

	return &Tracer{tracer, closer}
}

//Trace request
func (t *Tracer) Trace(req interfaces.Request) {
	var span opentracing.Span

	ctx, _ := tracer.Extract(opentracing.TextMap, opentracing.HTTPHeadersCarrier(req.GetTracerData()))
	span = tracer.StartSpan(req.GetPath(), opentracing.ChildOf(ctx))
	ext.SpanKindRPCClient.Set(span)

	headers := opentracing.HTTPHeadersCarrier(map[string][]string{})

	tracer.Inject(span.Context(), opentracing.TextMap, headers)

	req.SetTracerData(headers)


	span.Finish()
}

func setVariables() {
	if Endpoint == "" {
		Endpoint = env.Get("ORION_TRACER_ENDPOINT", "http://localhost:9411/api/v1/spans")
	}
	if HostAddr == "" {
		HostAddr = env.Get("ORION_TRACER_HOST_ADDR", "0.0.0.0:0")
	}
	e := env.Get("TRACER_ENABLED", "")
	if e == "1" || e == "true" {
		enabled = true
	}
}
