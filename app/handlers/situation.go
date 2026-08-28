package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	situationview "github.com/SkyvisorInsights/Aviation-tracker/app/view/situation"
	"github.com/SkyvisorInsights/skyvisor-go-shared/apiclient"
)

// SituationPage renders the global situation monitor.
//
// The route sits behind RequireAuth, so reaching this handler means the visitor
// has a session. As on the globe, a missing API token renders the page with an
// explanation rather than redirecting: /login is guest-only and would bounce an
// authenticated visitor straight back out with nothing said.
func (h *Handler) SituationPage(w http.ResponseWriter, r *http.Request) error {
	view := h.buildSituationView(r)
	view.News = h.buildNewsView(r)
	page := situationview.Page(view)
	return h.CreateCanvasLayout(w, r, "Global situation", page).Render(r.Context(), w)
}

func (h *Handler) buildSituationView(r *http.Request) situationview.View {
	client := h.service.API()
	if client == nil {
		return situationview.View{Unavailable: "Live situation data is not configured for this environment."}
	}

	accessToken, err := h.apiAccessToken(r)
	if err != nil || accessToken == "" {
		return situationview.View{Unavailable: "Sign in again to load live situation data."}
	}

	layers, err := client.SituationLayers(r.Context(), accessToken)
	if err != nil {
		slog.WarnContext(r.Context(), "situation catalogue unavailable", "error", err)
		// A degraded page that says what is missing beats an error page: every
		// other part of this view is still meaningful without the catalogue.
		switch {
		case errors.Is(err, apiclient.ErrUnauthorized):
			return situationview.View{Unavailable: "Sign in again to load live situation data."}
		default:
			return situationview.View{Unavailable: "Situation monitoring is unavailable right now."}
		}
	}
	return situationview.View{Layers: layers}
}

// SituationLayerData proxies one layer's GeoJSON from the API.
//
// The browser cannot call the API directly — the access token lives in a
// server-side session — so this is the BFF's job. The body is passed through
// untouched: it goes straight into a MapLibre source, and re-encoding it here
// would only add a place for the two to disagree.
func (h *Handler) SituationLayerData(w http.ResponseWriter, r *http.Request) error {
	client := h.service.API()
	if client == nil {
		http.Error(w, "situation data is not configured", http.StatusServiceUnavailable)
		return nil
	}

	accessToken, err := h.apiAccessToken(r)
	if err != nil || accessToken == "" {
		http.Error(w, "sign in again", http.StatusUnauthorized)
		return nil
	}

	layerID := strings.TrimSuffix(chi.URLParam(r, "layer"), ".geojson")
	body, err := client.SituationLayerGeoJSON(r.Context(), accessToken, layerID, apiclient.SituationQuery{
		BBox: r.URL.Query().Get("bbox"),
	})
	switch {
	case errors.Is(err, apiclient.ErrPaymentRequired):
		// The rail already renders locked layers as an upgrade link, so this is
		// only reachable by a hand-made request. Answer honestly and cheaply.
		http.Error(w, "this layer requires a higher plan", http.StatusPaymentRequired)
		return nil
	case errors.Is(err, apiclient.ErrUnauthorized):
		http.Error(w, "sign in again", http.StatusUnauthorized)
		return nil
	case err != nil:
		slog.WarnContext(r.Context(), "situation layer unavailable", "error", err, "layer", layerID)
		http.Error(w, "layer is unavailable", http.StatusBadGateway)
		return nil
	}

	w.Header().Set("Content-Type", "application/geo+json")
	w.Header().Set("Cache-Control", "private, max-age=30")
	_, writeErr := w.Write(body)
	return writeErr
}

// SituationNews renders the news rail, as a partial.
//
// Its own endpoint rather than part of the page bootstrap: the rail pages, it
// refreshes on its own cadence, and inlining sixty stories would inflate every
// page load with content most visitors never scroll to.
func (h *Handler) SituationNews(w http.ResponseWriter, r *http.Request) error {
	view := h.buildNewsView(r)
	return situationview.NewsRail(view).Render(r.Context(), w)
}

func (h *Handler) buildNewsView(r *http.Request) situationview.NewsView {
	client := h.service.API()
	if client == nil {
		return situationview.NewsView{Unavailable: "Live news is not configured for this environment."}
	}

	accessToken, err := h.apiAccessToken(r)
	if err != nil || accessToken == "" {
		return situationview.NewsView{Unavailable: "Sign in again to load live news."}
	}

	page, err := client.SituationNews(r.Context(), accessToken, apiclient.SituationNewsQuery{
		Cursor:    strings.TrimSpace(r.URL.Query().Get("cursor")),
		Languages: splitFilter(r.URL.Query().Get("languages")),
		Countries: splitFilter(r.URL.Query().Get("countries")),
	})
	if err != nil {
		// A rail that says why it is empty beats an error page: the globe
		// beside it is still working.
		slog.WarnContext(r.Context(), "situation news unavailable", "error", err)
		return situationview.NewsView{Unavailable: "News is unavailable right now."}
	}

	return situationview.NewsView{Items: page.Data, NextCursor: page.NextCursor}
}

// splitFilter reads a comma-separated filter, dropping empties so a trailing
// comma is not sent on as a filter matching nothing.
func splitFilter(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if cleaned := strings.TrimSpace(part); cleaned != "" {
			values = append(values, cleaned)
		}
	}
	return values
}
