package globe

import (
	"encoding/json"
	"testing"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
)

// testLookup knows a handful of airports so the unresolved path is exercised.
func testLookup(iata string) (float64, float64, bool) {
	switch iata {
	case "LIS":
		return 38.7813, -9.1359, true
	case "JFK":
		return 40.6413, -73.7781, true
	case "HND":
		return 35.5494, 139.7798, true
	case "SFO":
		return 37.6188, -122.375, true
	default:
		return 0, 0, false
	}
}

func TestRiskBucketBoundaries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		onTime   float64
		avgDelay float64
		flights  int
		want     string
	}{
		{"no flights is not a failure", 0, 0, 0, RiskOK},
		{"healthy", 95, 4, 100, RiskOK},
		{"exactly at watch on-time boundary", 80, 0, 100, RiskOK},
		{"just below watch on-time boundary", 79.9, 0, 100, RiskWatch},
		{"exactly at watch delay boundary", 100, 15, 100, RiskWatch},
		{"just below watch delay boundary", 100, 14.9, 100, RiskOK},
		{"exactly at risk on-time boundary", 60, 0, 100, RiskWatch},
		{"just below risk on-time boundary", 59.9, 0, 100, RiskRisk},
		{"exactly at risk delay boundary", 100, 30, 100, RiskRisk},
		{"just below risk delay boundary", 100, 29.9, 100, RiskWatch},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RiskBucket(tc.onTime, tc.avgDelay, tc.flights); got != tc.want {
				t.Fatalf("RiskBucket(%v, %v, %d) = %q, want %q", tc.onTime, tc.avgDelay, tc.flights, got, tc.want)
			}
		})
	}
}

func TestBuildRoutesResolvesAndReportsUnresolved(t *testing.T) {
	t.Parallel()
	routes := []apiclient.RouteStat{
		{DepartureIATA: "LIS", ArrivalIATA: "JFK", Flights: 40, OnTimePercent: 90, AvgDelayMinutes: 5},
		{DepartureIATA: "lis", ArrivalIATA: "ZZZ", Flights: 10}, // arrival unknown
		{DepartureIATA: "QQQ", ArrivalIATA: "JFK", Flights: 10}, // departure unknown
		{DepartureIATA: "LIS", ArrivalIATA: "LIS", Flights: 10}, // degenerate
		{DepartureIATA: "", ArrivalIATA: "JFK", Flights: 10},    // blank
	}

	collection, unresolved := BuildRoutes(routes, testLookup, "")

	if len(collection.Features) != 1 {
		t.Fatalf("features = %d, want 1", len(collection.Features))
	}
	if got := collection.Features[0].Properties["id"]; got != "LIS-JFK" {
		t.Fatalf("id = %v, want LIS-JFK", got)
	}
	if len(unresolved) != 2 {
		t.Fatalf("unresolved = %v, want [ZZZ QQQ]", unresolved)
	}
}

func TestBuildRoutesFiltersByRisk(t *testing.T) {
	t.Parallel()
	routes := []apiclient.RouteStat{
		{DepartureIATA: "LIS", ArrivalIATA: "JFK", Flights: 40, OnTimePercent: 95, AvgDelayMinutes: 2},
		{DepartureIATA: "HND", ArrivalIATA: "SFO", Flights: 40, OnTimePercent: 20, AvgDelayMinutes: 60},
	}

	all, _ := BuildRoutes(routes, testLookup, "")
	if len(all.Features) != 2 {
		t.Fatalf("unfiltered features = %d, want 2", len(all.Features))
	}

	risky, _ := BuildRoutes(routes, testLookup, RiskRisk)
	if len(risky.Features) != 1 {
		t.Fatalf("risk-filtered features = %d, want 1", len(risky.Features))
	}
	if got := risky.Features[0].Properties["risk"]; got != RiskRisk {
		t.Fatalf("risk = %v", got)
	}
}

func TestBuildRoutesWithoutLookup(t *testing.T) {
	t.Parallel()
	collection, unresolved := BuildRoutes([]apiclient.RouteStat{{DepartureIATA: "LIS", ArrivalIATA: "JFK"}}, nil, "")
	if len(collection.Features) != 0 || len(unresolved) != 0 {
		t.Fatal("a nil lookup must produce nothing rather than guessing")
	}
	// An empty collection must still serialise as a FeatureCollection with an
	// array, not a null, or the browser's setData throws.
	encoded, _ := json.Marshal(collection)
	if string(encoded) != `{"type":"FeatureCollection","features":[]}` {
		t.Fatalf("empty collection = %s", encoded)
	}
}

func TestRouteFeatureGeometryTypes(t *testing.T) {
	t.Parallel()
	atlantic := RouteFeature(lis, jfk, nil)
	if atlantic.Geometry.Type != "LineString" {
		t.Fatalf("Atlantic route geometry = %q, want LineString", atlantic.Geometry.Type)
	}

	pacific := RouteFeature(hnd, sfo, nil)
	if pacific.Geometry.Type != "MultiLineString" {
		t.Fatalf("Pacific route geometry = %q, want MultiLineString", pacific.Geometry.Type)
	}
	parts, ok := pacific.Geometry.Coordinates.([][][]float64)
	if !ok || len(parts) != 2 {
		t.Fatalf("Pacific route should have 2 parts, got %T %v", pacific.Geometry.Coordinates, ok)
	}
}

// The JS layer expressions read these property names. A rename here breaks the
// map silently, so this test is the contract between the two languages.
func TestRoutePropertyNamesAreStable(t *testing.T) {
	t.Parallel()
	collection, _ := BuildRoutes([]apiclient.RouteStat{
		{DepartureIATA: "LIS", ArrivalIATA: "JFK", Flights: 40, OnTimePercent: 90, AvgDelayMinutes: 5},
	}, testLookup, "")

	encoded, err := json.Marshal(collection.Features[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := []string{"id", "dep", "arr", "flights", "avg_delay", "on_time_pct", "risk"}
	if len(decoded.Properties) != len(want) {
		t.Fatalf("property set = %v, want exactly %v", decoded.Properties, want)
	}
	for _, key := range want {
		if _, ok := decoded.Properties[key]; !ok {
			t.Fatalf("missing property %q; the map bundle reads this name", key)
		}
	}
}

func TestBuildHubsScalesTraffic(t *testing.T) {
	t.Parallel()
	collection, unresolved := BuildHubs([]apiclient.AirportStat{
		{IATA: "LIS", Departures: 30, Arrivals: 20, AvgDelayMinutes: 4},
		{IATA: "ZZZ", Departures: 5},
	}, testLookup)

	if len(collection.Features) != 1 {
		t.Fatalf("features = %d, want 1", len(collection.Features))
	}
	if got := collection.Features[0].Properties["traffic"]; got != 50 {
		t.Fatalf("traffic = %v, want 50", got)
	}
	if len(unresolved) != 1 || unresolved[0] != "ZZZ" {
		t.Fatalf("unresolved = %v", unresolved)
	}
}

// Low-sample breakdowns must omit their ratios from the payload entirely, not
// merely hide them in CSS.
func TestBuildTrustHubsOmitsLowSampleRatios(t *testing.T) {
	t.Parallel()
	collection, _ := BuildTrustHubs([]apiclient.TrustBreakdown{
		{Dimension: "airport", Value: "LIS", Evaluated: 40, Precision: 0.9, FalsePositiveRate: 0.1},
		{Dimension: "airport", Value: "JFK", Evaluated: 5, Precision: 0.5, FalsePositiveRate: 0.5},
		{Dimension: "airline", Value: "TP", Evaluated: 90},
	}, testLookup, 20)

	if len(collection.Features) != 2 {
		t.Fatalf("features = %d, want 2 (airline dimension has no geometry)", len(collection.Features))
	}

	byIATA := map[string]Feature{}
	for _, feature := range collection.Features {
		byIATA[feature.Properties["iata"].(string)] = feature
	}

	high := byIATA["LIS"]
	if _, ok := high.Properties["precision"]; !ok {
		t.Fatal("well-sampled breakdown should carry precision")
	}
	if _, ok := high.Properties["low_sample"]; ok {
		t.Fatal("well-sampled breakdown must not be flagged low_sample")
	}

	low := byIATA["JFK"]
	if low.Properties["low_sample"] != true {
		t.Fatal("under-sampled breakdown should be flagged")
	}
	encoded, _ := json.Marshal(low)
	for _, forbidden := range []string{"precision", "false_positive_rate"} {
		if containsKey(encoded, forbidden) {
			t.Fatalf("low-sample payload leaks %q: %s", forbidden, encoded)
		}
	}
}

func containsKey(payload []byte, key string) bool {
	var decoded struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return false
	}
	_, ok := decoded.Properties[key]
	return ok
}

func TestParseRoute(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, dep, arr string }{
		{"LIS → LHR", "LIS", "LHR"},
		{"lis->lhr", "LIS", "LHR"},
		{"LIS-LHR", "LIS", "LHR"},
		{"  MEM → CDG  ", "MEM", "CDG"},
	}
	for _, tc := range cases {
		dep, arr, ok := ParseRoute(tc.in)
		if !ok || dep != tc.dep || arr != tc.arr {
			t.Fatalf("ParseRoute(%q) = (%q, %q, %v)", tc.in, dep, arr, ok)
		}
	}

	for _, bad := range []string{"", "   ", "LIS"} {
		if _, _, ok := ParseRoute(bad); ok {
			t.Fatalf("ParseRoute(%q) should fail", bad)
		}
	}
}

func TestMergeUnresolvedDeduplicatesAndSorts(t *testing.T) {
	t.Parallel()
	got := MergeUnresolved([]string{"ZZZ", "AAA"}, []string{"AAA", "MMM"}, nil, []string{""})
	if got.Count != 3 {
		t.Fatalf("count = %d, want 3", got.Count)
	}
	want := []string{"AAA", "MMM", "ZZZ"}
	for i, code := range want {
		if got.Codes[i] != code {
			t.Fatalf("codes = %v, want %v", got.Codes, want)
		}
	}
}

func TestCountRisk(t *testing.T) {
	t.Parallel()
	collection, _ := BuildRoutes([]apiclient.RouteStat{
		{DepartureIATA: "LIS", ArrivalIATA: "JFK", Flights: 10, OnTimePercent: 99, AvgDelayMinutes: 1},
		{DepartureIATA: "HND", ArrivalIATA: "SFO", Flights: 10, OnTimePercent: 10, AvgDelayMinutes: 90},
	}, testLookup, "")

	if got := CountRisk(collection, RiskRisk); got != 1 {
		t.Fatalf("CountRisk(risk) = %d, want 1", got)
	}
	if got := CountRisk(collection, RiskOK); got != 1 {
		t.Fatalf("CountRisk(ok) = %d, want 1", got)
	}
}

// Averaging longitudes arithmetically breaks across the antimeridian: the mean
// of -179 and 179 is 0 the naive way, when it should be 180.
func TestMeanLongitudeHandlesAntimeridian(t *testing.T) {
	t.Parallel()
	collection := FeatureCollection{Type: "FeatureCollection", Features: []Feature{
		{Geometry: Geometry{Type: "Point", Coordinates: []float64{-179, 0}}},
		{Geometry: Geometry{Type: "Point", Coordinates: []float64{179, 0}}},
	}}

	lon, ok := MeanLongitude(collection)
	if !ok {
		t.Fatal("MeanLongitude returned not-ok for populated input")
	}
	if abs(abs(lon)-180) > 0.001 {
		t.Fatalf("MeanLongitude = %v, want ±180", lon)
	}

	if _, ok := MeanLongitude(FeatureCollection{}); ok {
		t.Fatal("empty collection should report no centre")
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
