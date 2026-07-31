package globe

import (
	"fmt"
	"math"
	"strings"

	geodata "github.com/SkyvisorInsights/Aviation-tracker/app/static/geo"
)

// Server-rendered fallback globe.
//
// This is the default render, not a degraded one: the page ships a real globe
// as inline SVG and the WebGL map replaces it once the bundle loads. That gives
// a no-JS globe, a no-WebGL globe, a printable view, a non-blank first paint,
// and a rendering path Go tests can assert on.
//
// The projection is orthographic, so the result is a sphere seen from space
// rather than a flat map — the same silhouette the WebGL globe presents.

const (
	svgSize   = 1000.0
	svgRadius = 460.0
	svgCenter = svgSize / 2
)

// Ortho projects lon/lat onto a sphere viewed from directly above
// (CenterLon, CenterLat).
type Ortho struct {
	CenterLon float64
	CenterLat float64
	Radius    float64
	OriginX   float64
	OriginY   float64
}

func NewOrtho(centerLon, centerLat float64) Ortho {
	return Ortho{
		CenterLon: centerLon,
		CenterLat: centerLat,
		Radius:    svgRadius,
		OriginX:   svgCenter,
		OriginY:   svgCenter,
	}
}

// Project maps a coordinate to SVG space. The bool reports whether the point is
// on the visible hemisphere; points on the far side must not be drawn.
func (o Ortho) Project(lon, lat float64) (float64, float64, bool) {
	latRad := lat * degToRad
	lonRad := (lon - o.CenterLon) * degToRad
	centerLatRad := o.CenterLat * degToRad

	cosC := math.Sin(centerLatRad)*math.Sin(latRad) +
		math.Cos(centerLatRad)*math.Cos(latRad)*math.Cos(lonRad)
	if cosC < 0 {
		return 0, 0, false
	}

	x := o.Radius * math.Cos(latRad) * math.Sin(lonRad)
	y := o.Radius * (math.Cos(centerLatRad)*math.Sin(latRad) -
		math.Sin(centerLatRad)*math.Cos(latRad)*math.Cos(lonRad))

	// SVG y grows downward.
	return o.OriginX + x, o.OriginY - y, true
}

// SVGGlobe is the data the fallback renders.
type SVGGlobe struct {
	CenterLon float64
	CenterLat float64
	Routes    FeatureCollection
	Hubs      FeatureCollection
}

func coordPair(value any) (float64, float64, bool) {
	pair, ok := value.([]float64)
	if !ok || len(pair) != 2 {
		return 0, 0, false
	}
	return pair[0], pair[1], true
}

func lineStrings(geometry Geometry) [][][]float64 {
	switch geometry.Type {
	case "LineString":
		if coords, ok := geometry.Coordinates.([][]float64); ok {
			return [][][]float64{coords}
		}
	case "MultiLineString":
		if coords, ok := geometry.Coordinates.([][][]float64); ok {
			return coords
		}
	}
	return nil
}

// path emits an SVG path for a run of coordinates, breaking the path wherever
// the line passes behind the sphere so it does not reappear as a chord.
func (o Ortho) path(coords [][]float64, builder *strings.Builder) {
	drawing := false
	for _, pair := range coords {
		if len(pair) != 2 {
			continue
		}
		x, y, visible := o.Project(pair[0], pair[1])
		if !visible {
			drawing = false
			continue
		}
		if drawing {
			fmt.Fprintf(builder, "L%.1f %.1f", x, y)
		} else {
			fmt.Fprintf(builder, "M%.1f %.1f", x, y)
			drawing = true
		}
	}
}

// LandPath builds the country outlines for the visible hemisphere.
func (o Ortho) LandPath() string {
	var builder strings.Builder
	for _, ring := range geodata.Land() {
		drawing := false
		for _, pair := range ring {
			x, y, visible := o.Project(pair[0], pair[1])
			if !visible {
				drawing = false
				continue
			}
			if drawing {
				fmt.Fprintf(&builder, "L%.1f %.1f", x, y)
			} else {
				fmt.Fprintf(&builder, "M%.1f %.1f", x, y)
				drawing = true
			}
		}
	}
	return builder.String()
}

// GraticulePath draws meridians and parallels every 30 degrees, sampled finely
// enough to curve correctly under the projection.
func (o Ortho) GraticulePath() string {
	var builder strings.Builder

	for lon := -180.0; lon < 180; lon += 30 {
		coords := make([][]float64, 0, 73)
		for lat := -90.0; lat <= 90; lat += 2.5 {
			coords = append(coords, []float64{lon, lat})
		}
		o.path(coords, &builder)
	}
	for lat := -60.0; lat <= 60; lat += 30 {
		coords := make([][]float64, 0, 145)
		for lon := -180.0; lon <= 180; lon += 2.5 {
			coords = append(coords, []float64{lon, lat})
		}
		o.path(coords, &builder)
	}

	return builder.String()
}

// RoutePath renders arcs for one risk bucket, so each bucket can be coloured
// with its own token.
func (o Ortho) RoutePath(routes FeatureCollection, risk string) string {
	var builder strings.Builder
	for _, feature := range routes.Features {
		if featureRisk(feature) != risk {
			continue
		}
		for _, line := range lineStrings(feature.Geometry) {
			o.path(line, &builder)
		}
	}
	return builder.String()
}

// SVGHub is a projected hub ready to render as a circle.
type SVGHub struct {
	IATA   string
	X      float64
	Y      float64
	Radius float64
	Risk   string
}

// HubCircles projects hubs, dropping any on the far side of the sphere.
func (o Ortho) HubCircles(hubs FeatureCollection) []SVGHub {
	out := make([]SVGHub, 0, len(hubs.Features))
	for _, feature := range hubs.Features {
		lon, lat, ok := coordPair(feature.Geometry.Coordinates)
		if !ok {
			continue
		}
		x, y, visible := o.Project(lon, lat)
		if !visible {
			continue
		}
		out = append(out, SVGHub{
			IATA:   propString(feature.Properties, "iata"),
			X:      x,
			Y:      y,
			Radius: hubRadius(propInt(feature.Properties, "traffic")),
			Risk:   featureRisk(feature),
		})
	}
	return out
}

// hubRadius scales on the square root of traffic so area, not radius, tracks
// volume. Clamped so a single-flight airport is still clickable and a mega-hub
// does not swallow its neighbours.
func hubRadius(traffic int) float64 {
	if traffic <= 0 {
		return 3
	}
	r := 3 + math.Sqrt(float64(traffic))*1.4
	return math.Min(18, r)
}

func featureRisk(feature Feature) string {
	if risk := propString(feature.Properties, "risk"); risk != "" {
		return risk
	}
	return RiskOK
}

func propString(properties map[string]any, key string) string {
	if properties == nil {
		return ""
	}
	if value, ok := properties[key].(string); ok {
		return value
	}
	return ""
}

func propInt(properties map[string]any, key string) int {
	if properties == nil {
		return 0
	}
	switch value := properties[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	}
	return 0
}

// SVGViewBox is the coordinate space the template renders into.
func SVGViewBox() string {
	return fmt.Sprintf("0 0 %.0f %.0f", svgSize, svgSize)
}

// SphereRadius and SphereCenter describe the globe body for the template.
func SphereRadius() float64 { return svgRadius }
func SphereCenter() float64 { return svgCenter }

// AltText describes the globe for screen readers and is regenerated on every
// data refresh, so it never drifts from what is drawn.
func AltText(stats Stats, unresolved Unresolved) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Globe showing %d %s between %d %s",
		stats.Routes, plural(stats.Routes, "route", "routes"),
		stats.Hubs, plural(stats.Hubs, "airport", "airports"))

	if stats.Flights > 0 {
		fmt.Fprintf(&builder, " and %d watched %s",
			stats.Flights, plural(stats.Flights, "flight", "flights"))
	}
	builder.WriteString(".")

	if stats.AtRisk > 0 {
		fmt.Fprintf(&builder, " %d %s at risk.", stats.AtRisk, plural(stats.AtRisk, "route is", "routes are"))
	}
	if unresolved.Count > 0 {
		fmt.Fprintf(&builder, " %d %s could not be placed: %s.",
			unresolved.Count,
			plural(unresolved.Count, "airport", "airports"),
			strings.Join(unresolved.Codes, ", "))
	}
	return builder.String()
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return singular
	}
	return pluralForm
}
