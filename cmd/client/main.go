// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/net/http/otelhttp/example/client/client.go

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := newExporter(ctx)
	if err != nil {
		return nil, err
	}

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, err
}

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx)
}

func main() {
	ctx := context.Background()

	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	url := flag.String("server", "http://localhost:3030/hello", "server url")
	flag.Parse()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	var body []byte
	var statusCode int

	tr := otel.Tracer("example-otel/cmd/client")
	err = func(ctx context.Context) error {
		ctx, span := tr.Start(ctx, "say hello")
		defer span.End()
		req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)

		fmt.Printf("Sending request...\n")
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		body, err = io.ReadAll(res.Body)
		_ = res.Body.Close()

		statusCode = res.StatusCode

		return err
	}(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response Received: %s\n", body)
	fmt.Printf("Response status: %d\n\n\n", statusCode)
	fmt.Printf("Waiting for few seconds to export spans ...\n\n")
	time.Sleep(10 * time.Second)
	fmt.Printf("Inspect traces on otlptracehttp endpoint\n")
}
