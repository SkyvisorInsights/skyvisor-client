package pricing

import (
	"context"
	"strings"
	"testing"
)

func TestPricingPageRenders(t *testing.T) {
	var sb strings.Builder
	if err := PricingPage().Render(context.Background(), &sb); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := sb.String()
	for _, want := range []string{
		"Watch one flight free",
		"$19", "$180", "$49", "$490", "Custom",
		"Most popular",
		"x-data=\"{ yearly: true }\"",
		"/register?plan=pro",
		"mailto:fernandocorreia316@gmail.com",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in rendered pricing page", want)
		}
	}
	if strings.Contains(html, "—") || strings.Contains(html, "–") {
		t.Error("em/en dash found in pricing page copy")
	}
}

func TestBusinessIsNotSelfServe(t *testing.T) {
	// Business used to link to /register?plan=business, which carried the
	// intent through signup to a checkout that has only one Stripe price: the
	// Pro one. A Business buyer was charged the Pro price and given Pro
	// entitlements, having been shown a $49 Business page.
	//
	// Business is sales-led until it has a price of its own, so the only thing
	// this asserts is that no self-serve path back to that bug exists.
	var sb strings.Builder
	if err := PricingPage().Render(context.Background(), &sb); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := sb.String()

	if strings.Contains(html, "plan=business") {
		t.Error("the pricing page still offers a self-serve Business checkout")
	}
	if !strings.Contains(html, "SkyVisor%20Business") {
		t.Error("Business has no contact route, so the tier cannot be bought at all")
	}
	// Pro is the one product that genuinely is self-serve.
	if !strings.Contains(html, "/register?plan=pro") {
		t.Error("Pro lost its self-serve checkout")
	}
}
