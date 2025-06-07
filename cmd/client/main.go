// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/net/http/otelhttp/example/client/client.go

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.1"
)

func initTracer() (*sdktrace.TracerProvider, error) {
	exp, err := newExporter()
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

func newExporter() (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(context.Background())
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tp, err := initTracer()
	if err != nil {
		logger.Error(fmt.Sprintf("error setting up OTel SDK - %e", err))
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

	err = func() error {
		req, _ := http.NewRequest("GET", *url, nil)

		logger.Info("Sending request...")
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
		logger.Error(err.Error())
	}

	logger.Info(fmt.Sprintf("Response Received: %s", body))
	logger.Info(fmt.Sprintf("Response status: %d", statusCode))
	fmt.Printf("Waiting for few seconds to export spans ...\n\n")
	time.Sleep(10 * time.Second)
	fmt.Printf("Inspect traces on otlptracehttp endpoint\n")
}
