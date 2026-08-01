package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	globeview "github.com/SkyvisorInsights/Aviation-tracker/app/view/globe"
	"golang.org/x/sync/errgroup"
)

const (
	globeDefaultWindowDays = 7
	globeMaxWindowDays     = 90
	globeUpstreamTimeout   = 20 * time.Second

	// Low-sample threshold mirrors domain.TrustLowSampleThreshold. Breakdowns
	// below it render without their ratios.
	globeLowSampleThreshold = 20
)

// globeDemoEnabled reports whether the globe may fall back to demo data.
//
// Off unless SKYVISOR_GLOBE_DEMO is explicitly set, and only ever consulted
// when there is no API session, so it can never mask or override real data.
// The page is required to label itself when this is in effect.
func globeDemoEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SKYVISOR_GLOBE_DEMO"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// GlobePage renders the authenticated Global View.
//
// The route sits behind RequireAuth, so reaching this handler means the visitor
// already has a session. Redirecting to /login when the API token is missing
// would bounce them straight back out again — /login is guest-only and sends
// authenticated users to the homepage — leaving them on a random page with no
// explanation. Render the globe and say what is unavailable instead.
func (h *Handler) GlobePage(w http.ResponseWriter, r *http.Request) error {
	accessToken, _ := h.apiAccessToken(r)

	view, err := h.buildGlobeView(r, accessToken)
	if err != nil {
		// A bad filter is the caller's mistake, not a server fault.
		if errors.Is(err, errInvalidWindow) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}
		return err
	}

	// Live refreshes swap the panels only, so the WebGL context behind them
	// survives the update.
	if globePartial(r) == "globe-panels" {
		return globeview.Panels(view).Render(r.Context(), w)
	}

	page := globeview.Page(view)
	if globePartial(r) == "globe" {
		return page.Render(r.Context(), w)
	}
	return h.CreateCanvasLayout(w, r, "Global view", page).Render(r.Context(), w)
}

// GlobeData serves the envelope as JSON, for the map bundle and for polling.
func (h *Handler) GlobeData(w http.ResponseWriter, r *http.Request) error {
	accessToken, err := h.apiAccessToken(r)
	if err != nil || accessToken == "" || h.service.API() == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return nil
	}

	view, err := h.buildGlobeView(r, accessToken)
	if err != nil {
		if errors.Is(err, errInvalidWindow) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Skyvisor-Upstream-Ms", strconv.FormatInt(view.Envelope.UpstreamMS, 10))
	return json.NewEncoder(w).Encode(view.Envelope)
}

// globePartial reports which fragment the caller wants, or "" for a full page.
func globePartial(r *http.Request) string {
	switch r.URL.Query().Get("partial") {
	case "globe-panels":
		return "globe-panels"
	case "globe":
		return "globe"
	}
	switch strings.TrimSpace(r.Header.Get("HX-Target")) {
	case "globe-panels", "#globe-panels":
		return "globe-panels"
	case "globe-view", "#globe-view":
		return "globe"
	}
	return ""
}

// parseWindowDays rejects malformed input rather than silently defaulting, so a
// typo in a shared link is visible instead of quietly changing the numbers.
func parseWindowDays(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return globeDefaultWindowDays, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 1 || value > globeMaxWindowDays {
		return 0, errInvalidWindow
	}
	return value, nil
}

var errInvalidWindow = &globeError{message: "window_days must be a whole number between 1 and 90"}

type globeError struct{ message string }

func (e *globeError) Error() string { return e.message }

func (h *Handler) buildGlobeView(r *http.Request, accessToken string) (globeview.View, error) {
	query := r.URL.Query()

	windowDays, err := parseWindowDays(query.Get("window_days"))
	if err != nil {
		return globeview.View{}, err
	}

	filters := globeview.Filters{
		Airline:    strings.ToUpper(strings.TrimSpace(query.Get("airline"))),
		Airport:    strings.ToUpper(strings.TrimSpace(query.Get("airport"))),
		Risk:       strings.ToLower(strings.TrimSpace(query.Get("risk"))),
		WindowDays: windowDays,
	}
	if filters.Risk != globeview.RiskOK && filters.Risk != globeview.RiskWatch && filters.Risk != globeview.RiskRisk {
		filters.Risk = ""
	}

	ctx, cancel := context.WithTimeout(r.Context(), globeUpstreamTimeout)
	defer cancel()

	var (
		analytics                             apiclient.AnalyticsReport
		dashboard                             apiclient.OperationsDashboard
		trust                                 apiclient.DecisionTrustMetrics
		hasAnalytics, hasOperations, hasTrust bool
	)

	// The three upstreams are independent, so fetch them concurrently and let
	// each one degrade on its own. A single failing panel must never take down
	// the page.
	started := time.Now()
	hasSession := accessToken != "" && h.service.API() != nil

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		if !hasSession {
			return nil
		}
		report, err := h.service.API().Analytics(groupCtx, accessToken, apiclient.AnalyticsQuery{
			Airline:    filters.Airline,
			Airport:    filters.Airport,
			WindowDays: windowDays,
		})
		if err != nil {
			slog.WarnContext(r.Context(), "globe analytics unavailable", "error", err)
			return nil
		}
		analytics, hasAnalytics = report, true
		return nil
	})

	group.Go(func() error {
		if !hasSession {
			return nil
		}
		report, err := h.service.API().OperationsDashboard(groupCtx, accessToken)
		if err != nil {
			slog.WarnContext(r.Context(), "globe operations unavailable", "error", err)
			return nil
		}
		dashboard, hasOperations = report, true
		return nil
	})

	group.Go(func() error {
		if !hasSession {
			return nil
		}
		metrics, err := h.service.API().DecisionTrustMetrics(groupCtx, accessToken)
		if err != nil {
			slog.WarnContext(r.Context(), "globe trust metrics unavailable", "error", err)
			return nil
		}
		trust, hasTrust = metrics, true
		return nil
	})

	_ = group.Wait()
	upstreamMS := time.Since(started).Milliseconds()

	// Local development without an API session: stand in demo data so the globe
	// can be worked on. Only when there is genuinely no session, and the view
	// carries the flag so the page states plainly that it is not real.
	demo := false
	if !hasSession && globeDemoEnabled() {
		demo = true
		analytics, hasAnalytics = globeview.DemoAnalytics(), true
		dashboard.Attention = globeview.DemoAttention()
		hasOperations = true
	}

	lookup := globeview.CoordLookup(h.service.CoordLookup())

	routes, routesUnresolved := globeview.BuildRoutes(analytics.Routes, lookup, filters.Risk)
	hubs, hubsUnresolved := globeview.BuildHubs(analytics.Airports, lookup)

	// Trust breakdowns add airports analytics may not have ranked, which is the
	// whole point of putting trust on a map.
	trustHubs, trustUnresolved := globeview.BuildTrustHubs(trust.Breakdowns, lookup, globeLowSampleThreshold)
	hubs = mergeHubs(hubs, trustHubs)

	markers := globeview.FleetMarkersFromWatches(dashboard.Watches, h.service.CoordLookup())
	flights := globeview.FleetMarkerPoints(markers)
	watched := len(dashboard.Watches)
	if demo {
		flights = globeview.DemoFlightPoints(lookup)
		watched = globeview.DemoWatchCount()
	}

	envelope := globeview.Envelope{
		GeneratedAt: time.Now().UTC(),
		WindowDays:  windowDays,
		Filters:     filters,
		UpstreamMS:  upstreamMS,
		Routes:      routes,
		Hubs:        hubs,
		Flights:     flights,
		Unresolved:  globeview.MergeUnresolved(routesUnresolved, hubsUnresolved, trustUnresolved),
		Stats: globeview.Stats{
			Routes:         len(routes.Features),
			Hubs:           len(hubs.Features),
			Flights:        len(flights.Features),
			WatchedFlights: watched,
			AtRisk:         globeview.CountRisk(routes, globeview.RiskRisk),
			OnTimePercent:  analytics.Summary.OnTimePercent,
			AvgDelay:       analytics.Summary.AvgDepartureDelayMinutes,
			SampleSize:     analytics.SampleSize,
			HasAnalytics:   hasAnalytics,
			HasOperations:  hasOperations,
			HasTrust:       hasTrust,
		},
	}

	message := ""
	switch {
	case demo:
		message = ""
	case !hasSession:
		message = "Your session has no API access. Sign in again to load live routes and flights."
	case !hasAnalytics && !hasOperations:
		message = "Live data is temporarily unavailable. The globe shows the basemap only."
	}

	return globeview.View{
		Envelope:      envelope,
		Demo:          demo,
		Message:       message,
		Attention:     globeview.AttentionRows(dashboard.Attention, h.service.CoordLookup()),
		Ortho:         globeview.NewOrtho(globeCenterLon(routes), 18),
		BootstrapJSON: globeview.MarshalBootstrap(envelope),
	}, nil
}

// globeCenterLon points the server-rendered globe at the data rather than
// always showing the Atlantic. Falls back to a view that shows most landmass.
func globeCenterLon(routes globeview.FeatureCollection) float64 {
	if lon, ok := globeview.MeanLongitude(routes); ok {
		return lon
	}
	return -20
}

func mergeHubs(primary, extra globeview.FeatureCollection) globeview.FeatureCollection {
	seen := make(map[string]struct{}, len(primary.Features))
	for _, feature := range primary.Features {
		if iata, ok := feature.Properties["iata"].(string); ok {
			seen[iata] = struct{}{}
		}
	}
	for _, feature := range extra.Features {
		iata, _ := feature.Properties["iata"].(string)
		if _, dup := seen[iata]; dup {
			continue
		}
		seen[iata] = struct{}{}
		primary.Features = append(primary.Features, feature)
	}
	return primary
}
