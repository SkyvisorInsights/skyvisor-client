package handlers

import (
	"net/http"

	"github.com/SkyvisorInsights/Aviation-tracker/app/view/pricing"
)

func (h *Handler) PricingPage(w http.ResponseWriter, r *http.Request) error {
	return h.CreateLayout(w, r, "Pricing", pricing.PricingPage()).Render(r.Context(), w)
}
