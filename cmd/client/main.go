// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/net/http/otelhttp/example/client/client.go

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.1"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exp, err := newTraceExporter(ctx)
	if err != nil {
		return nil, err
	}

	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("example-otel/client"),
	)

	bsp := sdktrace.NewBatchSpanProcessor(exp)

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func initLogger(ctx context.Context) (*slog.Logger, error) {
	exp, err := newLogExporter(ctx)
	if err != nil {
		return nil, err
	}

	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("example-otel/client"),
	)

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
	)

	logger := otelslog.NewLogger("example-otel/client", otelslog.WithLoggerProvider(lp))
	return logger, nil
}

func newTraceExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx)
}

func newLogExporter(ctx context.Context) (sdklog.Exporter, error) {
	return otlploghttp.New(ctx)
}

func main() {
	ctx := context.Background()
	logger, err := initLogger(ctx)
	if err != nil {
		panic(fmt.Sprintf("error setting up OTel Log SDK - %v", err))
	}

	tp, err := initTracer(ctx)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("error setting up OTel Trace SDK - %e", err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %e", err))
		}
	}()

	url := flag.String("server", "http://localhost:3030/hello", "server url")
	flag.Parse()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	var body []byte
	var statusCode int

	tr := otel.Tracer("example-otel/cmd/client")
	err = func() error {
		ctx, span := tr.Start(context.Background(), "Start request")
		defer span.End()
		req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)

		logger.InfoContext(ctx, "Sending request...")
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		body, err = io.ReadAll(res.Body)
		_ = res.Body.Close()

		statusCode = res.StatusCode

		return err
	}()
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
	}

	logger.InfoContext(ctx, fmt.Sprintf("Response Received: %s", body))
	logger.InfoContext(ctx, fmt.Sprintf("Response status: %d", statusCode))
	fmt.Printf("Waiting for few seconds to export spans ...\n\n")
	time.Sleep(10 * time.Second)
	fmt.Printf("Inspect traces on otlptracehttp endpoint\n")
}
