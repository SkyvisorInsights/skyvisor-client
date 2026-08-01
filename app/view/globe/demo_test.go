package globe

import (
	"encoding/json"
	"strings"
	"testing"
)

// Demo data must behave exactly like real data: same resolver, same dropping of
// unknown airports, same property names the map layers read.

func TestDemoFlightsUseResolvedCoordinates(t *testing.T) {
	t.Parallel()
	flights := DemoFlightPoints(testLookup)

	if len(flights.Features) == 0 {
		t.Fatal("demo produced no flights")
	}
	for _, feature := range flights.Features {
		coords, ok := feature.Geometry.Coordinates.([]float64)
		if !ok || len(coords) != 2 {
			t.Fatalf("bad geometry: %#v", feature.Geometry)
		}
		if coords[0] == 0 && coords[1] == 0 {
			t.Fatalf("demo flight %v placed at 0,0", feature.Properties["id"])
		}
		if coords[1] < -90 || coords[1] > 90 {
			t.Fatalf("latitude out of range: %v", coords[1])
		}
	}
}

// The lookup here knows only four airports, so most demo flights must be
// dropped rather than invented.
func TestDemoFlightsDropUnresolvableRoutes(t *testing.T) {
	t.Parallel()
	full := DemoFlightPoints(testLookup)
	none := DemoFlightPoints(func(string) (float64, float64, bool) { return 0, 0, false })

	if len(none.Features) != 0 {
		t.Fatalf("with nothing resolvable the demo should draw nothing, got %d", len(none.Features))
	}
	if len(full.Features) >= DemoWatchCount() {
		t.Fatalf("expected some demo flights to be dropped by a partial lookup, got %d of %d", len(full.Features), DemoWatchCount())
	}
	if nilLookup := DemoFlightPoints(nil); len(nilLookup.Features) != 0 {
		t.Fatal("a nil lookup must produce nothing")
	}
}

// The map layers read these names; a rename breaks the globe silently.
func TestDemoFlightPropertiesMatchLayerExpectations(t *testing.T) {
	t.Parallel()
	flights := DemoFlightPoints(testLookup)

	encoded, err := json.Marshal(flights.Features[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"id", "callsign", "delay_minutes", "risk"} {
		if _, ok := decoded.Properties[key]; !ok {
			t.Fatalf("demo flight missing %q, which the map layers read", key)
		}
	}
}

func TestDemoFlightRiskTracksDelay(t *testing.T) {
	t.Parallel()
	flights := DemoFlightPoints(testLookup)

	for _, feature := range flights.Features {
		delay := propInt(feature.Properties, "delay_minutes")
		risk := propString(feature.Properties, "risk")
		switch {
		case delay >= 30 && risk != RiskRisk:
			t.Fatalf("%v: %dm delay should be at risk, got %q", feature.Properties["id"], delay, risk)
		case delay >= 15 && delay < 30 && risk != RiskWatch:
			t.Fatalf("%v: %dm delay should be watch, got %q", feature.Properties["id"], delay, risk)
		case delay < 15 && risk != RiskOK:
			t.Fatalf("%v: %dm delay should be ok, got %q", feature.Properties["id"], delay, risk)
		}
	}
}

// At least one demo aircraft must be stale, so the dimmed-and-frozen path is
// exercised rather than only ever seen in production.
func TestDemoIncludesAStaleFlight(t *testing.T) {
	t.Parallel()
	flights := DemoFlightPoints(func(iata string) (float64, float64, bool) {
		// Resolve everything so the whole demo set is present.
		return 10, 10.5, true
	})

	stale := 0
	for _, feature := range flights.Features {
		if feature.Properties["stale"] == true {
			stale++
		}
	}
	if stale == 0 {
		t.Fatal("no demo flight is marked stale")
	}
}

// The demo route set must include an unresolvable airport so the "airports
// hidden" disclosure is exercised.
func TestDemoRoutesExerciseTheUnresolvedFootnote(t *testing.T) {
	t.Parallel()
	_, unresolved := BuildRoutes(DemoAnalytics().Routes, testLookup, "")
	if len(unresolved) == 0 {
		t.Fatal("demo routes should include at least one unresolvable airport")
	}

	note := UnresolvedNote(MergeUnresolved(unresolved))
	if !strings.Contains(note, "hidden") {
		t.Fatalf("unresolved note = %q", note)
	}
}

func TestDemoAnalyticsIsInternallyConsistent(t *testing.T) {
	t.Parallel()
	report := DemoAnalytics()

	if report.Summary.OnTimePercent <= 0 || report.Summary.OnTimePercent > 100 {
		t.Fatalf("on-time percent out of range: %v", report.Summary.OnTimePercent)
	}
	if len(report.Routes) == 0 || len(report.Airports) == 0 {
		t.Fatal("demo analytics is empty")
	}

	// Every route endpoint should appear in the airport list, except the
	// deliberately unresolvable one.
	airports := map[string]bool{}
	for _, a := range report.Airports {
		airports[a.IATA] = true
	}
	for _, r := range report.Routes {
		if r.ArrivalIATA == "ZZZ" {
			continue
		}
		if !airports[r.DepartureIATA] {
			t.Fatalf("route origin %s has no airport entry", r.DepartureIATA)
		}
	}
}

func TestDemoAttentionRowsResolve(t *testing.T) {
	t.Parallel()
	rows := AttentionRows(DemoAttention(), testLookup)

	if len(rows) != len(DemoAttention()) {
		t.Fatalf("rows = %d, want %d — unresolvable rows must still be listed", len(rows), len(DemoAttention()))
	}
	positioned := 0
	for _, row := range rows {
		if row.HasPoint {
			positioned++
		}
		if row.Dep == "" || row.Arr == "" {
			t.Fatalf("row %s did not parse its route %q", row.ID, row.Route)
		}
	}
	if positioned == 0 {
		t.Fatal("no demo attention row resolved to a position")
	}
}
