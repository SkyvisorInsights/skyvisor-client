package handlers

import (
	"log/slog"
	"net/http"

	"github.com/SkyvisorInsights/Aviation-tracker/app/view/legal"
	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

func (h *Handler) TermsPage(w http.ResponseWriter, r *http.Request) error {
	return h.CreateLayout(w, r, "Terms of Service", legal.TermsPage()).Render(r.Context(), w)
}

func (h *Handler) PrivacyPage(w http.ResponseWriter, r *http.Request) error {
	return h.CreateLayout(w, r, "Privacy Policy", legal.PrivacyPage()).Render(r.Context(), w)
}

// DataSourcesPage credits every upstream the situation monitor draws on.
//
// Built from the live catalogue rather than a hand-maintained list: several of
// these sources permit reuse only with attribution, and a page that has to be
// updated by hand is one that eventually is not.
func (h *Handler) DataSourcesPage(w http.ResponseWriter, r *http.Request) error {
	var layers []domain.SituationLayer

	// Public page, so there is no session to read the catalogue with. It is
	// the same list for everyone — entitlement changes what a caller may read,
	// not who must be credited.
	if client := h.service.API(); client != nil {
		if accessToken, err := h.apiAccessToken(r); err == nil && accessToken != "" {
			if fetched, err := client.SituationLayers(r.Context(), accessToken); err == nil {
				layers = fetched
			} else {
				slog.WarnContext(r.Context(), "data sources catalogue unavailable", "error", err)
			}
		}
	}

	return h.CreateLayout(w, r, "Data sources", legal.DataSourcesPage(layers)).Render(r.Context(), w)
}
