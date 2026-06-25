package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		os.Exit(1)
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", portNum)

	client := http.Client{Timeout: 2 * time.Second}
	// #nosec G704 -- URL is restricted to localhost with validated numeric port.
	resp, err := client.Get(healthURL)
	if err != nil {
		os.Exit(1)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			os.Exit(1)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
