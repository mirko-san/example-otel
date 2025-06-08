// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/net/http/otelhttp/example/server/server.go

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

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

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// Implement an HTTP Handler func to be instrumented
func helloHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.InfoContext(ctx, "Received request")
		fmt.Fprintf(w, "Hello, World")
	}
}

func errorHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.InfoContext(ctx, "Received request")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func httpbinHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.InfoContext(ctx, "Received request")
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
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exp, err := newTraceExporter(ctx)
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

func initLogger(ctx context.Context) (*slog.Logger, error) {
	exp, err := newLogExporter(ctx)
	if err != nil {
		return nil, err
	}

	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("example-otel/server"),
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
	serverPort := getEnv("EXAMPLE_SERVER_PORT", "3030")

	tp, err := initTracer(ctx)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("error setting up OTel Trace SDK - %e", err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %e", err))
		}
	}()

	// Initialize HTTP handler instrumentation
	mux := http.NewServeMux()

	mux.Handle("/hello", otelhttp.NewHandler(http.HandlerFunc(helloHandler(logger)), "hello"))
	mux.Handle("/error", otelhttp.NewHandler(http.HandlerFunc(errorHandler(logger)), "error"))
	mux.Handle("/httpbin/", otelhttp.NewHandler(http.HandlerFunc(httpbinHandler(logger)), "httpbin"))
	err = http.ListenAndServe(fmt.Sprintf(":%s", serverPort), mux)
	if err != nil {
		logger.Error(err.Error())
	}
}
