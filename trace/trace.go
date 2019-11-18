package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kostiamol/centerms/cfg"

	"github.com/kostiamol/centerms/log"

	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"google.golang.org/grpc/metadata"
)

// Init initializes jaeger trace agent.
func Init(serviceName string, a cfg.TraceAgent) error {
	exp, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: fmt.Sprintf("%s:%d", a.Addr.Host, a.Addr.Port),
		Process: jaeger.Process{
			ServiceName: serviceName,
		},
	})
	if err != nil {
		return fmt.Errorf("func NewExporter: %s", err)
	}
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		return fmt.Errorf("func Register: %s", err)
	}

	trace.RegisterExporter(exp)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	return nil
}

// GetSpanFromTrace .
func GetSpanFromTrace(l log.Logger, traceSpan, name string) (context.Context, *trace.Span) {
	if traceSpan == "" {
		l.Info("trace span is empty, new trace span was created")
		return trace.StartSpan(context.Background(), name)
	}

	var spanCtx trace.SpanContext
	err := json.Unmarshal([]byte(traceSpan), &spanCtx)
	if err != nil {
		l.Errorf("trace span %s is not valid, new trace span was created", traceSpan)
		return trace.StartSpan(context.Background(), name)
	}

	ctx, span := trace.StartSpanWithRemoteParent(context.Background(), name, spanCtx)

	return ctx, span
}

// GetTraceString .
func GetTraceString(ctx trace.SpanContext) string {
	b, err := json.Marshal(ctx)
	if err != nil {
		return ""
	}

	return string(b)
}

// SpanFromReqAPI creates span from request with b3 propagation.
func SpanFromReqAPI(r *http.Request, name string) (context.Context, *trace.Span) {
	name = fmt.Sprintf("api: %s", name)
	f := b3.HTTPFormat{}
	ctx, _ := f.SpanContextFromRequest(r)

	return trace.StartSpanWithRemoteParent(r.Context(), name, ctx)
}

// ClientSpanFromReqHTTP creates span with request b3 propagation.
func ClientSpanFromReqHTTP(ctx context.Context, r *http.Request, name string) (context.Context, *trace.Span) {
	ctx, span := trace.StartSpan(ctx, fmt.Sprintf("http_client: %s", name))
	span.AddAttributes(
		trace.StringAttribute("url", r.URL.String()),
		trace.StringAttribute("method", r.Method))

	f := b3.HTTPFormat{}
	f.SpanContextToRequest(span.SpanContext(), r)

	return ctx, span
}

// ClientSpanFromCtxGRPC creates span with grpc context propagation.
func ClientSpanFromCtxGRPC(ctx context.Context, name string) (context.Context, *trace.Span) {
	ctx, span := trace.StartSpan(ctx, fmt.Sprintf("grpc_client: %s", name))

	traceCtxBinary := propagation.Binary(span.SpanContext())
	ctx = metadata.AppendToOutgoingContext(ctx, "grpc-trace-bin", string(traceCtxBinary))

	return ctx, span
}

// ServerSpanFromCtxGRPC creates span with grpc context propagation.
func ServerSpanFromCtxGRPC(ctx context.Context, name string) (context.Context, *trace.Span) {
	spanCtx := trace.SpanContext{}
	name = fmt.Sprintf("grpc_server: %s", name)

	b := grpcGetTraceByteFromCtx(ctx)
	if b == nil {
		return trace.StartSpanWithRemoteParent(ctx, name, spanCtx)
	}

	if s, ok := propagation.FromBinary(b); ok {
		spanCtx = s
	}

	return trace.StartSpanWithRemoteParent(ctx, name, spanCtx)
}

func grpcGetTraceByteFromCtx(ctx context.Context) []byte {
	cmap, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}

	v, ok := cmap["grpc-trace-bin"]
	if !ok {
		return nil
	}

	if len(v) == 0 {
		return nil
	}

	return []byte(v[len(v)-1])
}
