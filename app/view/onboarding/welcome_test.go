package onboarding

import (
	"context"
	"strings"
	"testing"
)

func renderWelcome(t *testing.T, plan string) string {
	t.Helper()
	var sb strings.Builder
	if err := WelcomePage("Ada", plan, "test-csrf").Render(context.Background(), &sb); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

func TestWelcomeOffersCheckoutForPro(t *testing.T) {
	html := renderWelcome(t, "pro")
	if !strings.Contains(html, "/billing/checkout") {
		t.Error("Pro lost its checkout prompt")
	}
	if !strings.Contains(html, "You picked Pro.") {
		t.Error("Pro intent is not acknowledged")
	}
}

func TestWelcomeNeverOffersCheckoutForBusiness(t *testing.T) {
	// /billing/checkout has one Stripe price and it is Pro's. Rendering this
	// form for a Business intent charges the Pro price and grants Pro, which is
	// the mischarge this whole change exists to close. Business is sales-led.
	html := renderWelcome(t, "business")
	if strings.Contains(html, "/billing/checkout") {
		t.Error("a Business signup is still offered the Pro checkout form")
	}
	if strings.Contains(html, "You picked Business.") {
		t.Error("the page still claims a Business selection it cannot fulfil")
	}
}

func TestWelcomeOffersNoCheckoutWithoutAPlan(t *testing.T) {
	if strings.Contains(renderWelcome(t, ""), "/billing/checkout") {
		t.Error("checkout offered with no pricing intent")
	}
}
