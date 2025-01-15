package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v81"
)

func main() {
	ntfyURL := os.Getenv("NTFY_URL")
	if ntfyURL == "" {
		fmt.Fprintf(os.Stderr, "NTFY_URL not set\n")
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		const MaxBodyBytes = int64(65536)
		req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		event := stripe.Event{}

		if err := json.Unmarshal(payload, &event); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse webhook body json: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Printf("Forwarding event type: %v\n", event.Type)
		err = sendToNTFY(ntfyURL, string(event.Type), "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to forward event to ntfy: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	fmt.Println("Listening on :3000")
	err := http.ListenAndServe(":3000", mux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to listen on port 8000: %v\n", err)
		os.Exit(1)
	}
}

func requiredEnvVar(name string) string {
	val := os.Getenv(name)
	if val == "" {
		fmt.Fprintf(os.Stderr, "Required environment variable %v not set\n", name)
		os.Exit(1)
	}
	return val
}

func sendToNTFY(ntfyURL, eventType, clickURL string) error {
	req, _ := http.NewRequest("POST", ntfyURL, strings.NewReader(eventType))
	req.Header.Set("Title", "Stripe Event")
	if clickURL != "" {
		req.Header.Set("Click", clickURL)
	}
	_, err := http.DefaultClient.Do(req)
	return err
}
