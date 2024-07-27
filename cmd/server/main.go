package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

	resp, err := http.Get(targetURL)
	if err != nil {
		http.Error(w, "Failed to fetch data from httpbin", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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
	serverPort := getEnv("EXAMPLE_SERVER_PORT", "3030")

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
	mux := http.NewServeMux()

	mux.Handle("/hello", otelhttp.NewHandler(http.HandlerFunc(helloHandler), "hello"))
	mux.Handle("/error", otelhttp.NewHandler(http.HandlerFunc(errorHandler), "error"))
	mux.Handle("/httpbin/", otelhttp.NewHandler(http.HandlerFunc(httpbinHandler), "httpbin"))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", serverPort), mux))
}
