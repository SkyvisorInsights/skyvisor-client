package globe

import (
	"math"
	"strings"
	"testing"

	geodata "github.com/SkyvisorInsights/Aviation-tracker/app/static/geo"
)

func TestOrthoProjectsCentreToMiddle(t *testing.T) {
	t.Parallel()
	o := NewOrtho(0, 0)

	x, y, visible := o.Project(0, 0)
	if !visible {
		t.Fatal("the centre of the visible hemisphere must be visible")
	}
	closeTo(t, x, SphereCenter(), 1e-6, "centre x")
	closeTo(t, y, SphereCenter(), 1e-6, "centre y")
}

func TestOrthoHidesFarHemisphere(t *testing.T) {
	t.Parallel()
	o := NewOrtho(0, 0)

	if _, _, visible := o.Project(180, 0); visible {
		t.Fatal("the antipode must not be drawn")
	}
	if _, _, visible := o.Project(120, 0); visible {
		t.Fatal("a point beyond 90 degrees must not be drawn")
	}
	if _, _, visible := o.Project(89, 0); !visible {
		t.Fatal("a point just inside 90 degrees should be drawn")
	}
}

func TestOrthoEdgePointsSitOnTheRim(t *testing.T) {
	t.Parallel()
	o := NewOrtho(0, 0)

	x, y, visible := o.Project(90, 0)
	if !visible {
		t.Fatal("the limb should be visible")
	}
	radius := math.Hypot(x-SphereCenter(), y-SphereCenter())
	closeTo(t, radius, SphereRadius(), 1e-6, "limb radius")
}

// North must be up: SVG y grows downward, so a northern point needs a smaller y.
func TestOrthoNorthIsUp(t *testing.T) {
	t.Parallel()
	o := NewOrtho(0, 0)

	_, northY, _ := o.Project(0, 45)
	_, southY, _ := o.Project(0, -45)
	if northY >= southY {
		t.Fatalf("north y (%v) should be above south y (%v)", northY, southY)
	}
}

func TestOrthoEastIsRight(t *testing.T) {
	t.Parallel()
	o := NewOrtho(0, 0)

	eastX, _, _ := o.Project(45, 0)
	westX, _, _ := o.Project(-45, 0)
	if eastX <= westX {
		t.Fatalf("east x (%v) should be right of west x (%v)", eastX, westX)
	}
}

func TestOrthoAllProjectedPointsStayInsideTheSphere(t *testing.T) {
	t.Parallel()
	o := NewOrtho(20, 10)

	for lon := -180.0; lon <= 180; lon += 7 {
		for lat := -90.0; lat <= 90; lat += 7 {
			x, y, visible := o.Project(lon, lat)
			if !visible {
				continue
			}
			radius := math.Hypot(x-SphereCenter(), y-SphereCenter())
			if radius > SphereRadius()+1e-6 {
				t.Fatalf("point (%v, %v) projected outside the sphere at radius %v", lon, lat, radius)
			}
		}
	}
}

func TestLandPathIsNonEmptyAndWellFormed(t *testing.T) {
	t.Parallel()
	if len(geodata.Land()) == 0 {
		t.Fatal("no land geometry was embedded; run `make basemap`")
	}

	path := NewOrtho(0, 20).LandPath()
	if path == "" {
		t.Fatal("LandPath produced nothing")
	}
	if !strings.HasPrefix(path, "M") {
		t.Fatalf("path must start with a move command, got %.20s", path)
	}
	if strings.Contains(path, "NaN") || strings.Contains(path, "Inf") {
		t.Fatal("path contains non-finite coordinates")
	}
}

// Rotating the globe must actually change what is drawn.
func TestLandPathChangesWithRotation(t *testing.T) {
	t.Parallel()
	atlantic := NewOrtho(-30, 20).LandPath()
	pacific := NewOrtho(150, 20).LandPath()
	if atlantic == pacific {
		t.Fatal("rotating the globe produced an identical path")
	}
}

func TestGraticulePathIsWellFormed(t *testing.T) {
	t.Parallel()
	path := NewOrtho(0, 0).GraticulePath()
	if path == "" {
		t.Fatal("GraticulePath produced nothing")
	}
	if strings.Contains(path, "NaN") {
		t.Fatal("graticule contains NaN")
	}
	// Meridians and parallels both contribute, so there must be many subpaths.
	if strings.Count(path, "M") < 10 {
		t.Fatalf("expected many subpaths, got %d", strings.Count(path, "M"))
	}
}

func TestRoutePathSplitsPerRiskBucket(t *testing.T) {
	t.Parallel()
	o := NewOrtho(-40, 30)
	routes := FeatureCollection{Type: "FeatureCollection", Features: []Feature{
		RouteFeature(lis, jfk, map[string]any{"risk": RiskOK}),
		RouteFeature(lis, jfk, map[string]any{"risk": RiskRisk}),
	}}

	if o.RoutePath(routes, RiskOK) == "" {
		t.Fatal("ok bucket produced no path")
	}
	if o.RoutePath(routes, RiskRisk) == "" {
		t.Fatal("risk bucket produced no path")
	}
	if o.RoutePath(routes, RiskWatch) != "" {
		t.Fatal("watch bucket should be empty")
	}
}

// A route on the far side of the globe must not be drawn as a chord.
func TestRoutePathOmitsHiddenRoutes(t *testing.T) {
	t.Parallel()
	o := NewOrtho(0, 0) // centred on the Gulf of Guinea; the Pacific is behind
	routes := FeatureCollection{Type: "FeatureCollection", Features: []Feature{
		RouteFeature(hnd, sfo, map[string]any{"risk": RiskOK}),
	}}

	if path := o.RoutePath(routes, RiskOK); path != "" {
		t.Fatalf("a route on the far hemisphere should not be drawn, got %.40s", path)
	}
}

func TestHubCirclesDropFarSideAndScale(t *testing.T) {
	t.Parallel()
	o := NewOrtho(-40, 30)
	hubs := FeatureCollection{Type: "FeatureCollection", Features: []Feature{
		{Geometry: Geometry{Type: "Point", Coordinates: []float64{-9.1359, 38.7813}}, Properties: map[string]any{"iata": "LIS", "traffic": 100, "risk": RiskOK}},
		{Geometry: Geometry{Type: "Point", Coordinates: []float64{139.7798, 35.5494}}, Properties: map[string]any{"iata": "HND", "traffic": 10}},
	}}

	circles := o.HubCircles(hubs)
	if len(circles) != 1 || circles[0].IATA != "LIS" {
		t.Fatalf("circles = %+v, want only LIS", circles)
	}
	if circles[0].Radius <= 3 {
		t.Fatalf("radius %v should scale with traffic", circles[0].Radius)
	}
}

func TestHubRadiusIsClamped(t *testing.T) {
	t.Parallel()
	if got := hubRadius(0); got != 3 {
		t.Fatalf("hubRadius(0) = %v, want the 3 minimum", got)
	}
	if got := hubRadius(1_000_000); got > 18 {
		t.Fatalf("hubRadius is unbounded at %v", got)
	}
	small, large := hubRadius(10), hubRadius(1000)
	if small >= large {
		t.Fatal("radius should grow with traffic")
	}
}

func TestAltTextDescribesWhatIsDrawn(t *testing.T) {
	t.Parallel()
	text := AltText(Stats{Routes: 12, Hubs: 7, Flights: 3, AtRisk: 2}, Unresolved{Count: 1, Codes: []string{"ZZZ"}})

	for _, want := range []string{"12 routes", "7 airports", "3 watched flights", "2 routes are at risk", "ZZZ"} {
		if !strings.Contains(text, want) {
			t.Fatalf("alt text %q missing %q", text, want)
		}
	}
}

func TestAltTextSingularForms(t *testing.T) {
	t.Parallel()
	text := AltText(Stats{Routes: 1, Hubs: 1, Flights: 1, AtRisk: 1}, Unresolved{})
	for _, want := range []string{"1 route ", "1 airport", "1 watched flight", "1 route is at risk"} {
		if !strings.Contains(text, want) {
			t.Fatalf("alt text %q missing %q", text, want)
		}
	}
}

func TestAltTextEmptyGlobe(t *testing.T) {
	t.Parallel()
	text := AltText(Stats{}, Unresolved{})
	if !strings.Contains(text, "0 routes") {
		t.Fatalf("empty globe alt text = %q", text)
	}
	if strings.Contains(text, "at risk") {
		t.Fatalf("empty globe should not mention risk: %q", text)
	}
}
