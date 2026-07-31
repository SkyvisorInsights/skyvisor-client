package globe

import "math"

// Great-circle geometry.
//
// This lives in Go rather than the browser so the route builder is a pure,
// table-testable function, and so the authenticated page, the public trust
// share and the marketing globe all produce byte-identical geometry from one
// implementation.

const degToRad = math.Pi / 180

// Point is a [longitude, latitude] pair in degrees.
type Point struct {
	Lon float64
	Lat float64
}

// Interpolate returns the point a given fraction along the great circle from a
// to b, using spherical linear interpolation.
//
// Interpolating in lon/lat space instead would bow the route the wrong way:
// Lisbon to New York would pass through 39.7N rather than the correct 44.5N,
// roughly 530km out.
func Interpolate(a, b Point, fraction float64) Point {
	fraction = math.Min(1, math.Max(0, fraction))

	lat1, lon1 := a.Lat*degToRad, a.Lon*degToRad
	lat2, lon2 := b.Lat*degToRad, b.Lon*degToRad

	// Angular distance between the two points.
	sinHalfLat := math.Sin((lat2 - lat1) / 2)
	sinHalfLon := math.Sin((lon2 - lon1) / 2)
	h := sinHalfLat*sinHalfLat + math.Cos(lat1)*math.Cos(lat2)*sinHalfLon*sinHalfLon
	d := 2 * math.Asin(math.Sqrt(math.Min(1, h)))

	// Guard with an epsilon rather than an exact zero. Floating-point
	// contraction can leave d a hair above zero for identical points, and the
	// slerp below divides by sin(d), which is unstable as d approaches zero.
	// At this threshold the two points are well under a micrometre apart.
	if d < 1e-12 {
		return a
	}

	sinD := math.Sin(d)
	af := math.Sin((1-fraction)*d) / sinD
	bf := math.Sin(fraction*d) / sinD

	x := af*math.Cos(lat1)*math.Cos(lon1) + bf*math.Cos(lat2)*math.Cos(lon2)
	y := af*math.Cos(lat1)*math.Sin(lon1) + bf*math.Cos(lat2)*math.Sin(lon2)
	z := af*math.Sin(lat1) + bf*math.Sin(lat2)

	return Point{
		Lat: math.Atan2(z, math.Sqrt(x*x+y*y)) / degToRad,
		Lon: math.Atan2(y, x) / degToRad,
	}
}

// ArcPoints samples the great circle between a and b.
func ArcPoints(a, b Point, steps int) []Point {
	if steps < 1 {
		steps = 1
	}
	points := make([]Point, 0, steps+1)
	for i := 0; i <= steps; i++ {
		points = append(points, Interpolate(a, b, float64(i)/float64(steps)))
	}
	return points
}

// SplitAntimeridian breaks a sampled arc wherever it crosses ±180°, so each
// part can be drawn without a line streaking back across the whole map.
//
// Longitudes are interpolated to the exact crossing and the parts are closed at
// ±180 so the two halves meet at the edge.
func SplitAntimeridian(points []Point) [][]Point {
	if len(points) < 2 {
		if len(points) == 0 {
			return nil
		}
		return [][]Point{points}
	}

	parts := make([][]Point, 0, 2)
	current := []Point{points[0]}

	for i := 1; i < len(points); i++ {
		prev, next := points[i-1], points[i]
		delta := next.Lon - prev.Lon

		if math.Abs(delta) <= 180 {
			current = append(current, next)
			continue
		}

		// Crossing. Work out the latitude at the seam and close both sides.
		sign := 1.0
		if delta > 0 {
			sign = -1.0
		}
		adjustedNextLon := next.Lon + 360*sign
		var t float64
		if adjustedNextLon != prev.Lon {
			t = (sign*180 - prev.Lon) / (adjustedNextLon - prev.Lon)
		}
		seamLat := prev.Lat + (next.Lat-prev.Lat)*t

		current = append(current, Point{Lon: sign * 180, Lat: seamLat})
		parts = append(parts, current)
		current = []Point{{Lon: -sign * 180, Lat: seamLat}, next}
	}

	parts = append(parts, current)
	return parts
}

// Arc returns the great circle between two points, split at the antimeridian.
func Arc(a, b Point, steps int) [][]Point {
	return SplitAntimeridian(ArcPoints(a, b, steps))
}
