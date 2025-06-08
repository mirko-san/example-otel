package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	url := flag.String("server", "http://localhost:3030/hello", "server url")
	flag.Parse()

	client := http.Client{Transport: http.DefaultTransport}

	var body []byte
	var statusCode int

	err := func() error {
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
}
