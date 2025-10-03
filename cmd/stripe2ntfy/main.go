package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v79/webhook"
)

type eventNotification struct {
	title    string
	body     string
	clickURL string
}

type ntfyConfig struct {
	URL      string
	Username string
	Password string
	Token    string
}

func main() {
	ntfyCfg := ntfyConfig{
		URL:      requiredEnvVar("NTFY_URL"),
		Username: os.Getenv("NTFY_USERNAME"),
		Password: os.Getenv("NTFY_PASSWORD"),
		Token:    os.Getenv("NTFY_TOKEN"),
	}
	webhookSecret := requiredEnvVar("STRIPE_WEBHOOK_SECRET")

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "127.0.0.1:3000"
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

		signature := req.Header.Get("Stripe-Signature")
		if signature == "" {
			fmt.Fprintf(os.Stderr, "Received request with no Stripe-Signature header\n")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		event, err := webhook.ConstructEventWithOptions(payload, signature, webhookSecret, webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		fmt.Printf("Received event type: %v\n", event.Type)
		notification := createNotification(event.Type, event.Data.Raw)

		err = sendNotification(ntfyCfg, event.Livemode, notification)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to forward event to ntfy: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("Listening on %s ...\n", listenAddr)
	err := http.ListenAndServe(listenAddr, mux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to listen on %s: %v\n", listenAddr, err)
		os.Exit(1)
	}
}

func sendNotification(config ntfyConfig, liveMode bool, notification eventNotification) error {
	body := notification.body
	if !liveMode {
		body += " [test mode]"
	}

	req, _ := http.NewRequest(http.MethodPost, config.URL, strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	if config.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))
	} else if config.Username != "" && config.Password != "" {
		auth := fmt.Sprintf("%s:%s", config.Username, config.Password)
		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth))))
	}

	req.Header.Set("Markdown", "yes")
	req.Header.Set("Title", notification.title)
	req.Header.Set("Icon", "https://play-lh.googleusercontent.com/2PS6w7uBztfuMys5fgodNkTwTOE6bLVB2cJYbu5GHlARAK36FzO5bUfMDP9cEJk__cE")
	if notification.clickURL != "" {
		req.Header.Set("Click", notification.clickURL)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, bodyErr := io.ReadAll(resp.Body)
		return errors.Join(fmt.Errorf("ntfy returned unexpected status code %d: %s", resp.StatusCode, body), bodyErr)
	}
	return nil
}
