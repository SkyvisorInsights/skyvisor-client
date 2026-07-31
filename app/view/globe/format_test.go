package globe

import "testing"

func TestCallsign(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{"TP1363", "TP-1363"},
		{"AA845", "AA-845"},
		{"4U9525", "4U-9525"}, // alphanumeric carrier code
		{"BA1", "BA-1"},       // single-digit number
		{"tp1363", "TP-1363"}, // normalised
		{"  TP1363 ", "TP-1363"},
		{"TP-1363", "TP-1363"}, // already formatted
		{"", ""},
		{"TP", "TP"},     // no number
		{"1363", "1363"}, // no carrier
		{"VERYLONGTHING", "VERYLONGTHING"},
		{"ABCD1234", "ABCD1234"}, // prefix too long to be a carrier code
	}

	for _, tc := range cases {
		if got := Callsign(tc.in); got != tc.want {
			t.Fatalf("Callsign(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Unknown must never render as zero. Confusing "no data" with "the value is
// zero" is how a dashboard starts lying.
func TestUnknownRendersAsEmDashNotZero(t *testing.T) {
	t.Parallel()
	if got := Count(0, false); got != "—" {
		t.Fatalf("Count(unknown) = %q, want an em dash", got)
	}
	if got := Count(0, true); got != "0" {
		t.Fatalf("Count(0, known) = %q, want 0", got)
	}
	if got := Percent(0, false); got != "—" {
		t.Fatalf("Percent(unknown) = %q", got)
	}
	if got := Percent(0, true); got != "0.0%" {
		t.Fatalf("Percent(0, known) = %q", got)
	}
	if got := Minutes(0, false); got != "—" {
		t.Fatalf("Minutes(unknown) = %q", got)
	}
	if got := Minutes(12.4, true); got != "12m" {
		t.Fatalf("Minutes(12.4) = %q", got)
	}
}

// A falling delay is good; a falling on-time rate is not. The same numeric
// movement means opposite things depending on the metric.
func TestDeltaDirectionDependsOnMetric(t *testing.T) {
	t.Parallel()
	rising := DeltaFrom(90, 80, true, true, false)
	if !rising.Good || rising.Label != "+10.0" {
		t.Fatalf("rising on-time = %+v, want good +10.0", rising)
	}

	fallingOnTime := DeltaFrom(70, 80, true, true, false)
	if fallingOnTime.Good {
		t.Fatal("a falling on-time rate must not be reported as good")
	}

	fallingDelay := DeltaFrom(10, 20, true, true, true)
	if !fallingDelay.Good {
		t.Fatal("a falling delay must be reported as good")
	}

	risingDelay := DeltaFrom(30, 20, true, true, true)
	if risingDelay.Good {
		t.Fatal("a rising delay must not be reported as good")
	}
}

// A delta against a window we never measured would be an invented number.
func TestDeltaRequiresBothWindows(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ hasCurrent, hasPrevious bool }{
		{false, true}, {true, false}, {false, false},
	} {
		got := DeltaFrom(90, 80, tc.hasCurrent, tc.hasPrevious, false)
		if got.Known || got.Label != "—" {
			t.Fatalf("DeltaFrom(hasCurrent=%v, hasPrevious=%v) = %+v, want unknown", tc.hasCurrent, tc.hasPrevious, got)
		}
	}
}

func TestDeltaZero(t *testing.T) {
	t.Parallel()
	got := DeltaFrom(80, 80, true, true, false)
	if !got.Known || got.Label != "0%" || !got.Good {
		t.Fatalf("zero delta = %+v", got)
	}
}

func TestRouteLabel(t *testing.T) {
	t.Parallel()
	if got := RouteLabel("lis", " jfk "); got != "LIS → JFK" {
		t.Fatalf("RouteLabel = %q", got)
	}
	if got := RouteLabel("", "JFK"); got != "—" {
		t.Fatalf("RouteLabel with missing origin = %q, want em dash", got)
	}
}

func TestRiskLabel(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		RiskOK:    "On track",
		RiskWatch: "Watch",
		RiskRisk:  "At risk",
		"":        "—",
		"bogus":   "—",
	}
	for in, want := range cases {
		if got := RiskLabel(in); got != want {
			t.Fatalf("RiskLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnresolvedNote(t *testing.T) {
	t.Parallel()
	if got := UnresolvedNote(Unresolved{}); got != "" {
		t.Fatalf("no unresolved codes should produce no footnote, got %q", got)
	}
	if got := UnresolvedNote(Unresolved{Count: 1, Codes: []string{"ZZZ"}}); got != "1 airport hidden — no coordinates for ZZZ." {
		t.Fatalf("singular note = %q", got)
	}
	if got := UnresolvedNote(Unresolved{Count: 2, Codes: []string{"ZZZ", "QQQ"}}); got != "2 airports hidden — no coordinates for ZZZ, QQQ." {
		t.Fatalf("plural note = %q", got)
	}
}
