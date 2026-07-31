package dashboard

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
)

func TestPageRendersOperationalEvidence(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	dashboard := apiclient.OperationsDashboard{
		Summary: apiclient.OperationsSummary{ActiveWatches: 1, FlightsAtRisk: 1},
		Attention: []apiclient.OperationsAttention{{
			ID: "watch:1", Severity: "high", Title: "FX100 is delayed 42 minutes",
			FlightNumber: "FX100", Route: "MEM → CDG", Reason: "Check the downstream handoff.",
		}},
		Watches: []apiclient.OperationsWatch{{
			WatchID: "1", FlightNumber: "FX100", Route: "MEM → CDG", Status: "active", FreshnessStatus: "fresh", UpdatedAt: now,
		}},
		Freshness:   apiclient.OperationsFreshness{Provider: "aviationstack", Status: "fresh", FreshRecords: 1, StaleAfterSeconds: 120, LatestObservationAt: now},
		GeneratedAt: now,
	}
	markers := FleetMarkersFrom(dashboard.Watches, stubLookup)
	var output bytes.Buffer
	if err := Page(dashboard, markers, "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render dashboard: %v", err)
	}
	html := output.String()
	for _, expected := range []string{"What needs attention now", "FX100 is delayed 42 minutes", "MEM → CDG", "aviationstack", "Live data current"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("dashboard HTML missing %q", expected)
		}
	}
}

func TestPageRendersHonestEmptyState(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	if err := Page(apiclient.OperationsDashboard{}, nil, "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render dashboard: %v", err)
	}
	html := output.String()
	if !strings.Contains(html, "No operational risks detected") || !strings.Contains(html, "Missing evidence is shown as unknown") {
		t.Fatalf("empty dashboard did not explain missing data: %s", html)
	}
}

// stubLookup knows MEM only, so tests can exercise both the resolved and the
// unresolved path without a database.
func stubLookup(iata string) (float64, float64, bool) {
	switch iata {
	case "MEM":
		return 35.0424, -89.9767, true
	case "LIS":
		return 38.7813, -9.1359, true
	default:
		return 0, 0, false
	}
}

func TestFleetMarkersFromDropsUnresolvableOrigins(t *testing.T) {
	t.Parallel()
	watches := []apiclient.OperationsWatch{
		{FlightNumber: "FX100", Route: "MEM → CDG"},
		{FlightNumber: "ZZ999", Route: "ZZZ → QQQ"}, // origin not in the table
		{FlightNumber: "NR001", Route: ""},          // no route at all
	}

	markers := FleetMarkersFrom(watches, stubLookup)

	if len(markers) != 1 {
		t.Fatalf("markers = %d, want 1 (unresolvable origins must be dropped, not plotted at 0,0): %+v", len(markers), markers)
	}
	if markers[0].IATA != "MEM" || markers[0].Flight != "FX100" {
		t.Fatalf("unexpected marker: %+v", markers[0])
	}
	if markers[0].Lat == 0 && markers[0].Lon == 0 {
		t.Fatal("resolved marker sits at 0,0")
	}
}

func TestFleetMarkersFromWithoutLookup(t *testing.T) {
	t.Parallel()
	watches := []apiclient.OperationsWatch{{FlightNumber: "FX100", Route: "MEM → CDG"}}
	if markers := FleetMarkersFrom(watches, nil); markers != nil {
		t.Fatalf("nil lookup should yield no markers, got %+v", markers)
	}
}

func TestWatchMarkersJSONCarriesCoordinates(t *testing.T) {
	t.Parallel()
	markers := FleetMarkersFrom([]apiclient.OperationsWatch{
		{FlightNumber: "FX100", Route: "MEM → CDG"},
	}, stubLookup)

	got := watchMarkersJSON(markers)
	for _, expected := range []string{`"iata":"MEM"`, `"flight":"FX100"`, `"lon":-89.9767`, `"lat":35.0424`} {
		if !strings.Contains(got, expected) {
			t.Fatalf("watchMarkersJSON() = %s, missing %s", got, expected)
		}
	}

	if empty := watchMarkersJSON(nil); empty != "[]" {
		t.Fatalf("watchMarkersJSON(nil) = %q, want []", empty)
	}
}

func TestFleetMarkerLonLatBlankWhenUnresolved(t *testing.T) {
	t.Parallel()
	markers := FleetMarkersFrom([]apiclient.OperationsWatch{
		{FlightNumber: "FX100", Route: "MEM → CDG"},
	}, stubLookup)

	if lon := fleetMarkerLon(markers, "FX100"); lon != "-89.9767" {
		t.Fatalf("fleetMarkerLon(FX100) = %q", lon)
	}
	if lat := fleetMarkerLat(markers, "FX100"); lat != "35.0424" {
		t.Fatalf("fleetMarkerLat(FX100) = %q", lat)
	}
	// An unknown flight must render an empty attribute so the filmstrip button
	// declines to move the map rather than flying it to Null Island.
	if lon := fleetMarkerLon(markers, "ZZ999"); lon != "" {
		t.Fatalf("fleetMarkerLon(unknown) = %q, want empty", lon)
	}
	if lat := fleetMarkerLat(markers, "ZZ999"); lat != "" {
		t.Fatalf("fleetMarkerLat(unknown) = %q, want empty", lat)
	}
}
