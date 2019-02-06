package tracer

import (
	"fmt"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"

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

	// collector for the tracer
	collector zipkin.Collector

	// tracer implementation
	tracer opentracing.Tracer
)

// Tracer for orion
type Tracer struct {
}

// Close req tracer
type Close = func()

func init() {
	setVariables()
}

// New zipkin tracer
func New(service string) Tracer {

	if enabled {

		c, err := zipkin.NewHTTPCollector(Endpoint)
		if err != nil {
			fmt.Printf("unable to create Zipkin HTTP collector: %+v\n", err)
			os.Exit(1)
		}
		collector = c

		recorder := zipkin.NewRecorder(collector, Debug, HostAddr, service)

		t, err := zipkin.NewTracer(
			recorder,
			zipkin.ClientServerSameSpan(SameSpan),
			zipkin.TraceID128Bit(TraceID128Bit),
		)
		if err != nil {
			fmt.Printf("unable to create Zipkin tracer: %+v\n", err)
			os.Exit(1)
		}

		tracer = t

		opentracing.InitGlobalTracer(tracer)

	}

	return Tracer{}
}

// Trace request
func (t Tracer) Trace(req interfaces.Request) Close {
	var span opentracing.Span
	if enabled {
		ctx, _ := tracer.Extract(opentracing.TextMap, opentracing.HTTPHeadersCarrier(req.GetTracerData()))
		span = tracer.StartSpan(req.GetPath(), opentracing.ChildOf(ctx))
		ext.SpanKindRPCClient.Set(span)

		headers := opentracing.HTTPHeadersCarrier(map[string][]string{})

		tracer.Inject(span.Context(), opentracing.TextMap, headers)

		req.SetTracerData(headers)
		req.SetID(headers["X-B3-Traceid"][0])
	}

	return func() {
		if enabled {
			span.Finish()
		}
	}
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
