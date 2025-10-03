package main

import (
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v79"
)

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
