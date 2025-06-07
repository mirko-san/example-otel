// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/net/http/otelhttp/example/server/server.go

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.1"
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// Implement an HTTP Handler func to be instrumented
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func httpbinHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/httpbin"):]
	targetURL := "https://httpbin.org/" + path

	resp, err := otelhttp.Get(r.Context(), targetURL)
	if err != nil {
		http.Error(w, "Failed to fetch data from httpbin", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func initTracer() (*sdktrace.TracerProvider, error) {
	exp, err := newExporter()
	if err != nil {
		return nil, err
	}

	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("example-otel/server"),
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
	serverPort := getEnv("EXAMPLE_SERVER_PORT", "3030")

	tp, err := initTracer()
	if err != nil {
		logger.Error(fmt.Sprintf("error setting up OTel SDK - %e", err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %e", err))
		}
	}()

	// Initialize HTTP handler instrumentation
	mux := http.NewServeMux()

	mux.Handle("/hello", otelhttp.NewHandler(http.HandlerFunc(helloHandler), "hello"))
	mux.Handle("/error", otelhttp.NewHandler(http.HandlerFunc(errorHandler), "error"))
	mux.Handle("/httpbin/", otelhttp.NewHandler(http.HandlerFunc(httpbinHandler), "httpbin"))
	err = http.ListenAndServe(fmt.Sprintf(":%s", serverPort), mux)
	if err != nil {
		logger.Error(err.Error())
	}
}
