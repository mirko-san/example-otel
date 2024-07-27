package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Implement an HTTP Handler func to be instrumented
func httpHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

// Wrap the HTTP handler func with OTel HTTP instrumentation
func wrapHandler() {
	handler := http.HandlerFunc(httpHandler)
	wrappedHandler := otelhttp.NewHandler(handler, "hello")
	http.Handle("/hello", wrappedHandler)
}

var tp *sdktrace.TracerProvider

// initTracer creates and registers trace provider instance.
func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	exp, err := newExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdouttrace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx)
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down tracer privider: %e", err)
		}
	}()

	// Initialize HTTP handler instrumentation
	wrapHandler()
	log.Fatal(http.ListenAndServe(":3030", nil))
}
