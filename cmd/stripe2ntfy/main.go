package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"
)

type eventNotification struct {
	title    string
	body     string
	clickURL string
}

func main() {
	ntfyURL := requiredEnvVar("NTFY_URL")
	webhookSecret := requiredEnvVar("STRIPE_WEBHOOK_SECRET")

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

		event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		fmt.Printf("Received event type: %v\n", event.Type)
		notification := createNotification(event.Type, event.Data.Raw)

		err = sendNotification(ntfyURL, event.Livemode, notification)
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

func formatCurrency(currency stripe.Currency, amount int64) string {
	switch currency {
	case stripe.CurrencyUSD:
		return fmt.Sprintf("€%.2f", float64(amount)/100)
	case stripe.CurrencyEUR:
		return fmt.Sprintf("$%.2f", float64(amount)/100)
	default:
		return fmt.Sprintf("%d %s", amount, currency)
	}
}

func createNotification(eventType stripe.EventType, eventData json.RawMessage) eventNotification {
	switch eventType {
	case stripe.EventTypePaymentIntentSucceeded:
		var body string
		var clickURL string

		var intent stripe.PaymentIntent
		err := json.Unmarshal(eventData, &intent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing payment intent: %v\n", err)
			body = "Payment intent could not be parsed"
		} else {
			amount := formatCurrency(intent.Currency, intent.Amount)
			body = fmt.Sprintf("Received %s", amount)
			clickURL = "https://dashboard.stripe.com/payments/" + intent.ID
		}

		return eventNotification{
			title:    "💰 Payment Succeeded",
			body:     body,
			clickURL: clickURL,
		}
	case stripe.EventTypeRadarEarlyFraudWarningCreated:
		var body string
		var clickURL string

		var warning stripe.RadarEarlyFraudWarning
		err := json.Unmarshal(eventData, &warning)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing radar warning: %v\n", err)
			body = "Radar warning could not be parsed"
		} else {
			var actionableStatus string
			if warning.Actionable {
				actionableStatus = "actionable"
			} else {
				actionableStatus = "inactionable"
			}

			amount := formatCurrency(warning.PaymentIntent.Currency, warning.PaymentIntent.Amount)
			body = fmt.Sprintf("For %s (%s)", amount, actionableStatus)
			clickURL = "https://dashboard.stripe.com/payments/" + warning.Charge.ID
		}

		return eventNotification{
			title:    "⚠️ Early Fraud Warning",
			body:     body,
			clickURL: clickURL,
		}
	case stripe.EventTypeChargeDisputeCreated:
		var body string
		var clickURL string

		var dispute stripe.Dispute
		err := json.Unmarshal(eventData, &dispute)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing dispute: %v\n", err)
			body = "Dispute could not be parsed"
		} else {
			amount := formatCurrency(dispute.Currency, dispute.Amount)
			body = fmt.Sprintf("For %s", amount)
			clickURL = "https://dashboard.stripe.com/payments/" + dispute.Charge.ID
		}

		return eventNotification{
			title:    "💀 New Dispute",
			body:     body,
			clickURL: clickURL,
		}
	default:
		return eventNotification{
			title: "❓ Unknown Stripe Event",
			body:  fmt.Sprintf("Type: %s", eventType),
		}
	}
}

func sendNotification(ntfyURL string, liveMode bool, notification eventNotification) error {
	body := notification.body
	if !liveMode {
		body += " [test mode]"
	}

	req, _ := http.NewRequest("POST", ntfyURL, strings.NewReader(body))
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

	if resp.StatusCode != 200 {
		body, bodyErr := io.ReadAll(resp.Body)
		return errors.Join(fmt.Errorf("ntfy returned unexpected status code %d: %s", resp.StatusCode, body), bodyErr)
	}
	return nil
}
