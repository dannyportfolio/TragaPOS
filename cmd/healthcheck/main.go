package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	url := "http://127.0.0.1:8080/healthz"
	if len(os.Args) > 1 && os.Args[1] != "" {
		url = os.Args[1]
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Healthcheck request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Healthcheck unhealthy, status code: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	os.Exit(0)
}
