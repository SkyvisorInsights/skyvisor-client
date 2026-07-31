package globe

import (
	"math"
	"testing"
)

const tolerance = 1e-6

func closeTo(t *testing.T, got, want, eps float64, label string) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Fatalf("%s = %v, want %v (±%v)", label, got, want, eps)
	}
}

var (
	lis = Point{Lon: -9.1359, Lat: 38.7813}
	jfk = Point{Lon: -73.7781, Lat: 40.6413}
	hnd = Point{Lon: 139.7798, Lat: 35.5494}
	sfo = Point{Lon: -122.375, Lat: 37.6188}
)

// The whole reason this code exists: interpolating in lon/lat space puts the
// Lisbon-New York midpoint at 39.71N, when the great-circle midpoint is 44.50N.
func TestInterpolateFollowsGreatCircleNotStraightLine(t *testing.T) {
	t.Parallel()
	mid := Interpolate(lis, jfk, 0.5)

	linearMidLat := (lis.Lat + jfk.Lat) / 2
	if mid.Lat <= linearMidLat+3 {
		t.Fatalf("midpoint latitude %v is not meaningfully north of the linear midpoint %v", mid.Lat, linearMidLat)
	}
	closeTo(t, mid.Lat, 44.502619, 1e-4, "midpoint lat")
	closeTo(t, mid.Lon, -40.968273, 1e-4, "midpoint lon")
}

// The Go implementation must agree with the browser's arc library, since the
// same route is drawn by both.
func TestInterpolateMatchesJSReference(t *testing.T) {
	t.Parallel()
	// Reference values produced by the `arc` npm package, which the map bundle
	// uses. See app/assets/map/great-circle.test.js.
	mid := Interpolate(lis, jfk, 0.5)
	closeTo(t, mid.Lon, -40.96827289204085, 1e-6, "lon vs JS reference")
	closeTo(t, mid.Lat, 44.502619153139236, 1e-6, "lat vs JS reference")
}

func TestInterpolateEndpointsAndClamping(t *testing.T) {
	t.Parallel()
	start := Interpolate(lis, jfk, 0)
	closeTo(t, start.Lon, lis.Lon, tolerance, "start lon")
	closeTo(t, start.Lat, lis.Lat, tolerance, "start lat")

	end := Interpolate(lis, jfk, 1)
	closeTo(t, end.Lon, jfk.Lon, tolerance, "end lon")
	closeTo(t, end.Lat, jfk.Lat, tolerance, "end lat")

	if Interpolate(lis, jfk, -5) != start {
		t.Fatal("negative fraction should clamp to the start")
	}
	if Interpolate(lis, jfk, 5) != end {
		t.Fatal("fraction above one should clamp to the end")
	}
}

func TestInterpolateIdenticalPoints(t *testing.T) {
	t.Parallel()
	if got := Interpolate(lis, lis, 0.5); got != lis {
		t.Fatalf("Interpolate on identical points = %+v, want %+v", got, lis)
	}
}

// A polar route bows over the top; the midpoint must be north of both endpoints.
func TestInterpolatePolarRoute(t *testing.T) {
	t.Parallel()
	oslo := Point{Lon: 11.1, Lat: 60.2}
	anchorage := Point{Lon: -149.9, Lat: 61.2}

	mid := Interpolate(oslo, anchorage, 0.5)
	if mid.Lat <= oslo.Lat || mid.Lat <= anchorage.Lat {
		t.Fatalf("polar midpoint lat %v should exceed both endpoints (%v, %v)", mid.Lat, oslo.Lat, anchorage.Lat)
	}
}

func TestArcPointsCount(t *testing.T) {
	t.Parallel()
	if got := len(ArcPoints(lis, jfk, 16)); got != 17 {
		t.Fatalf("ArcPoints(16) returned %d points, want 17", got)
	}
	// Degenerate step counts must not panic or return nothing.
	if got := len(ArcPoints(lis, jfk, 0)); got != 2 {
		t.Fatalf("ArcPoints(0) returned %d points, want 2", got)
	}
}

func TestArcSplitsAtAntimeridian(t *testing.T) {
	t.Parallel()
	parts := Arc(hnd, sfo, 64)
	if len(parts) != 2 {
		t.Fatalf("Tokyo->San Francisco split into %d parts, want 2", len(parts))
	}

	// Each part ends (or starts) exactly at the seam.
	first, second := parts[0], parts[1]
	closeTo(t, math.Abs(first[len(first)-1].Lon), 180, tolerance, "first part ends at seam")
	closeTo(t, math.Abs(second[0].Lon), 180, tolerance, "second part starts at seam")

	// The seam latitude must be continuous across the break, otherwise the two
	// halves visibly fail to meet at the edge of the map.
	closeTo(t, first[len(first)-1].Lat, second[0].Lat, tolerance, "seam latitude continuity")

	// The two halves must be on opposite sides.
	if first[len(first)-1].Lon*second[0].Lon > 0 {
		t.Fatal("seam points should have opposite signs")
	}
}

func TestArcDoesNotSplitWhenNotCrossing(t *testing.T) {
	t.Parallel()
	if parts := Arc(lis, jfk, 64); len(parts) != 1 {
		t.Fatalf("Lisbon->New York split into %d parts, want 1", len(parts))
	}
}

// No segment may jump more than 180 degrees, which is what produces a line
// streaking back across the whole map.
func TestArcPartsHaveNoLongitudeJumps(t *testing.T) {
	t.Parallel()
	for _, pair := range [][2]Point{{hnd, sfo}, {lis, jfk}, {{Lon: 179, Lat: 10}, {Lon: -179, Lat: -10}}} {
		for _, part := range Arc(pair[0], pair[1], 64) {
			for i := 1; i < len(part); i++ {
				if delta := math.Abs(part[i].Lon - part[i-1].Lon); delta > 180 {
					t.Fatalf("segment jumps %v degrees of longitude", delta)
				}
			}
		}
	}
}

func TestSplitAntimeridianEdgeCases(t *testing.T) {
	t.Parallel()
	if got := SplitAntimeridian(nil); got != nil {
		t.Fatalf("SplitAntimeridian(nil) = %v, want nil", got)
	}
	single := []Point{lis}
	if got := SplitAntimeridian(single); len(got) != 1 || len(got[0]) != 1 {
		t.Fatalf("single point should pass through, got %v", got)
	}
}
