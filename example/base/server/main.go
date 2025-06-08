package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
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

		resp, err := http.Get(targetURL)
		if err != nil {
			http.Error(w, "Failed to fetch data from httpbin", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	serverPort := getEnv("EXAMPLE_SERVER_PORT", "3030")

	// Initialize HTTP handler instrumentation
	mux := http.NewServeMux()

	mux.Handle("/hello", helloHandler(logger))
	mux.Handle("/error", errorHandler(logger))
	mux.Handle("/httpbin/", httpbinHandler(logger))
	err := http.ListenAndServe(fmt.Sprintf(":%s", serverPort), mux)
	if err != nil {
		logger.ErrorContext(ctx, err.Error())
	}
}
