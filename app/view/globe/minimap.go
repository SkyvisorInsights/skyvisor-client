package globe

import (
	"fmt"
	"strings"

	geodata "github.com/SkyvisorInsights/Aviation-tracker/app/static/geo"
)

// Minimap inset.
//
// Deliberately a server-rendered SVG rather than a second MapLibre instance: a
// live minimap means a second WebGL context, a second tile cache and a second
// render loop, which on a phone is the difference between smooth and not. The
// only thing that moves at runtime is a small marker showing where the globe is
// pointed, which is a two-line transform in the browser.
//
// The projection is equirectangular, so lon/lat map linearly to x/y and the
// browser needs no projection maths of its own.

const (
	miniWidth  = 200.0
	miniHeight = 100.0
)

// MiniViewBox is the coordinate space of the inset.
func MiniViewBox() string {
	return fmt.Sprintf("0 0 %.0f %.0f", miniWidth, miniHeight)
}

func MiniWidth() float64  { return miniWidth }
func MiniHeight() float64 { return miniHeight }

// MiniProject maps a coordinate into the inset's space.
func MiniProject(lon, lat float64) (float64, float64) {
	x := (lon + 180) / 360 * miniWidth
	y := (90 - lat) / 180 * miniHeight
	return x, y
}

// MiniLandPath renders the world outline once, at build-of-page time.
//
// Rings that cross the antimeridian are split, otherwise a country spanning the
// seam is drawn as a band straight across the map.
func MiniLandPath() string {
	var builder strings.Builder

	for _, ring := range geodata.Land() {
		drawing := false
		var prevLon float64

		for i, pair := range ring {
			lon, lat := pair[0], pair[1]

			if i > 0 && absFloat(lon-prevLon) > 180 {
				// Seam crossing: break the path rather than drawing across it.
				drawing = false
			}
			prevLon = lon

			x, y := MiniProject(lon, lat)
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

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// FormatLonLat renders a coordinate for the readout, matching the fixed width
// the design uses so the value does not jitter as the cursor moves.
func FormatLonLat(value float64) string {
	return fmt.Sprintf("%.6f", value)
}
