package main

import (
	"fmt"
	"net/http"
)

func handleHeartbeat(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func pinger(port string) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%s/healthz", port), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned non-http.StatusOK status code")
	}

	return nil
}
