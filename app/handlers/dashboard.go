package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	dashboardview "github.com/SkyvisorInsights/Aviation-tracker/app/view/dashboard"
)

// DashboardPage is the API-backed signed-in command center.
func (h *Handler) DashboardPage(w http.ResponseWriter, r *http.Request) error {
	accessToken, err := h.apiAccessToken(r)
	if err != nil || accessToken == "" || h.service.API() == nil {
		http.Redirect(w, r, "/login?return_to=/dashboard", http.StatusSeeOther)
		return nil
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	dashboard, err := h.service.API().OperationsDashboard(ctx, accessToken)
	message := ""
	if err != nil {
		message = "Live operations are temporarily unavailable. Your existing watches and trips are unchanged."
		dashboard = apiclient.OperationsDashboard{GeneratedAt: time.Now().UTC()}
	}
	// Resolve watch origins to positions here; the browser has no airport table.
	markers := dashboardview.FleetMarkersFrom(dashboard.Watches, h.service.CoordLookup())
	page := dashboardview.Page(dashboard, markers, message)
	if r.Header.Get("HX-Request") == "true" && r.URL.Query().Get("partial") == "dashboard" {
		return page.Render(r.Context(), w)
	}
	return h.CreateLayout(w, r, "Operations dashboard", page).Render(r.Context(), w)
}
