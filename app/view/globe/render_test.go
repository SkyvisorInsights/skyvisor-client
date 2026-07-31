package globe

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
)

func sampleView(t *testing.T) View {
	t.Helper()

	routes, routesUnresolved := BuildRoutes([]apiclient.RouteStat{
		{DepartureIATA: "LIS", ArrivalIATA: "JFK", Flights: 42, OnTimePercent: 91, AvgDelayMinutes: 6},
		{DepartureIATA: "HND", ArrivalIATA: "SFO", Flights: 18, OnTimePercent: 41, AvgDelayMinutes: 55},
		{DepartureIATA: "LIS", ArrivalIATA: "ZZZ", Flights: 3},
	}, testLookup, "")

	hubs, hubsUnresolved := BuildHubs([]apiclient.AirportStat{
		{IATA: "LIS", Departures: 60, Arrivals: 55, AvgDelayMinutes: 6},
		{IATA: "JFK", Departures: 90, Arrivals: 88, AvgDelayMinutes: 12},
	}, testLookup)

	envelope := Envelope{
		WindowDays: 7,
		Routes:     routes,
		Hubs:       hubs,
		Flights:    emptyCollection(),
		Unresolved: MergeUnresolved(routesUnresolved, hubsUnresolved),
		Stats: Stats{
			Routes:         len(routes.Features),
			Hubs:           len(hubs.Features),
			WatchedFlights: 2,
			AtRisk:         CountRisk(routes, RiskRisk),
			OnTimePercent:  88.5,
			AvgDelay:       9,
			SampleSize:     640,
			HasAnalytics:   true,
			HasOperations:  true,
		},
	}

	return View{
		Envelope: envelope,
		Ortho:    NewOrtho(-40, 20),
		Attention: AttentionRows([]apiclient.OperationsAttention{
			{ID: "watch:1", Severity: "high", Title: "TP1363 delayed", FlightNumber: "TP1363", Route: "LIS → JFK", Reason: "Inbound aircraft is late."},
			{ID: "watch:2", Severity: "low", Title: "NH7 on time", FlightNumber: "NH7", Route: "HND → SFO", Reason: "No action needed."},
		}, testLookup),
		BootstrapJSON: MarshalBootstrap(envelope),
	}
}

func render(t *testing.T, v View) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Page(v).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render globe page: %v", err)
	}
	return buf.String()
}

// The globe must be a real, complete render with no JavaScript involved.
func TestPageRendersServerSideGlobe(t *testing.T) {
	t.Parallel()
	html := render(t, sampleView(t))

	if !strings.Contains(html, "<svg") || !strings.Contains(html, `data-globe-svg`) {
		t.Fatal("page did not render an inline SVG globe")
	}
	// Land, graticule and arcs must all be present as path data.
	if strings.Count(html, "<path") < 4 {
		t.Fatalf("expected land, graticule and per-risk arc paths, found %d paths", strings.Count(html, "<path"))
	}
	if !strings.Contains(html, "var(--map-land)") || !strings.Contains(html, "var(--map-ocean)") {
		t.Fatal("globe is not painted from the map design tokens")
	}
	// Hub circles carry a title so hovering names the airport.
	if !strings.Contains(html, "<title>LIS</title>") {
		t.Fatal("hub circles are missing accessible titles")
	}
	if strings.Contains(html, "NaN") {
		t.Fatal("rendered SVG contains NaN coordinates")
	}
}

func TestPageGlobeHasAccessibleDescription(t *testing.T) {
	t.Parallel()
	html := render(t, sampleView(t))

	if !strings.Contains(html, `role="img"`) {
		t.Fatal("the globe should expose itself as an image, not an unlabelled graphic")
	}
	for _, fragment := range []string{"2 routes", "2 airports", "1 route is at risk"} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("aria-label missing %q", fragment)
		}
	}
}

// The drawer is the accessible equivalent of the canvas, so every row must be
// reachable by keyboard and describe what is drawn.
func TestPageDrawerIsKeyboardReachable(t *testing.T) {
	t.Parallel()
	html := render(t, sampleView(t))

	if strings.Count(html, `tabindex="0"`) < 2 {
		t.Fatal("attention rows should be focusable")
	}
	if !strings.Contains(html, `data-globe-lon=`) {
		t.Fatal("rows with a known position should carry coordinates so focusing them can move the globe")
	}
	if !strings.Contains(html, "TP-1363") {
		t.Fatal("flight numbers should render as callsigns")
	}
	if !strings.Contains(html, "LIS → JFK") {
		t.Fatal("routes should render in the drawer")
	}
}

// Missing upstreams must read as unknown, never as a confident zero.
func TestPageDegradesHonestlyWithoutUpstreams(t *testing.T) {
	t.Parallel()
	v := View{
		Envelope: Envelope{
			WindowDays: 7,
			Routes:     emptyCollection(),
			Hubs:       emptyCollection(),
			Flights:    emptyCollection(),
		},
		Ortho:   NewOrtho(-20, 18),
		Message: "Live data is temporarily unavailable. The globe shows the basemap only.",
	}
	html := render(t, v)

	if !strings.Contains(html, "—") {
		t.Fatal("unknown statistics should render as em dashes")
	}
	if !strings.Contains(html, "Analytics are unavailable") {
		t.Fatal("the rail should say why the numbers are missing")
	}
	if !strings.Contains(html, "Live operations unavailable") {
		t.Fatal("the header should not claim live tracking when there is none")
	}
	// The basemap still renders, so the page is never blank.
	if !strings.Contains(html, "var(--map-land)") {
		t.Fatal("the basemap should render even with no data")
	}
}

func TestPageShowsUnresolvedFootnote(t *testing.T) {
	t.Parallel()
	html := render(t, sampleView(t))
	if !strings.Contains(html, "ZZZ") || !strings.Contains(html, "hidden") {
		t.Fatal("routes dropped for missing coordinates should be disclosed, not silently omitted")
	}
}

// The bootstrap payload is embedded in a script tag, so it must not be able to
// close that tag.
func TestBootstrapJSONIsScriptSafe(t *testing.T) {
	t.Parallel()
	envelope := Envelope{
		Routes:  emptyCollection(),
		Hubs:    emptyCollection(),
		Flights: emptyCollection(),
		Filters: Filters{Airline: "</script><script>alert(1)</script>"},
	}

	payload := MarshalBootstrap(envelope)
	if strings.Contains(payload, "</script>") {
		t.Fatalf("bootstrap payload can break out of its script tag: %s", payload)
	}

	html := render(t, View{Envelope: envelope, Ortho: NewOrtho(0, 0), BootstrapJSON: payload})
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatal("injected markup survived into the page")
	}
}

func TestPageRendersBootstrapForTheMapBundle(t *testing.T) {
	t.Parallel()
	html := render(t, sampleView(t))
	if !strings.Contains(html, `id="globe-bootstrap"`) {
		t.Fatal("the map bundle needs an embedded envelope to hydrate from")
	}
	if !strings.Contains(html, `type="application/json"`) {
		t.Fatal("bootstrap must be inert JSON, not executable script")
	}
}
