package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v79"
)

func createNotification(eventType stripe.EventType, eventData json.RawMessage) eventNotification {
	var body string
	var clickURL string

	switch eventType {

	case stripe.EventTypePaymentIntentSucceeded:
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

	case stripe.EventTypeCustomerSubscriptionCreated:
		var sub stripe.Subscription
		err := json.Unmarshal(eventData, &sub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing subscription: %v\n", err)
			body = "Subscription could not be parsed"
		} else {
			var details string
			if sub.Items != nil && len(sub.Items.Data) > 0 {
				item := sub.Items.Data[0]
				if item.Price != nil && item.Price.Recurring != nil {
					amount := formatCurrency(item.Price.Currency, item.Quantity*item.Price.UnitAmount)
					details = fmt.Sprintf("(%s/%s)", amount, item.Price.Recurring.Interval)
				}
			}

			body = fmt.Sprintf("For customer `%s` %s", sub.Customer.ID, details)
			clickURL = "https://dashboard.stripe.com/subscriptions/" + sub.ID
		}

		return eventNotification{
			title:    "🎉 New Subscription",
			body:     body,
			clickURL: clickURL,
		}

	case stripe.EventTypeCustomerSubscriptionDeleted:
		var sub stripe.Subscription
		err := json.Unmarshal(eventData, &sub)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing subscription: %v\n", err)
			body = "Subscription could not be parsed"
		} else {
			var details string
			if sub.Items != nil && len(sub.Items.Data) > 0 {
				item := sub.Items.Data[0]
				if item.Price != nil && item.Price.Recurring != nil {
					amount := formatCurrency(item.Price.Currency, item.Quantity*item.Price.UnitAmount)
					details = fmt.Sprintf("(%s/%s)", amount, item.Price.Recurring.Interval)
				}
			}
			body = fmt.Sprintf("Canceled for customer `%s` %s", sub.Customer.ID, details)
			clickURL = "https://dashboard.stripe.com/subscriptions/" + sub.ID
		}

		return eventNotification{
			title:    "😥 Subscription canceled",
			body:     body,
			clickURL: clickURL,
		}

	default:
		return eventNotification{
			title: "❓ Unknown Stripe Event",
			body:  fmt.Sprintf("Type: `%s`", eventType),
		}
	}
}
