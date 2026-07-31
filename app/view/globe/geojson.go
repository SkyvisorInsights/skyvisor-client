package globe

import (
	"sort"
	"strings"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
)

// Risk buckets. A route is judged on both punctuality and average delay, so a
// route that is usually on time but badly late when it slips still surfaces.
const (
	RiskOK    = "ok"
	RiskWatch = "watch"
	RiskRisk  = "risk"

	riskOnTimePercent   = 60.0
	riskAvgDelayMinutes = 30.0

	watchOnTimePercent   = 80.0
	watchAvgDelayMinutes = 15.0

	// arcSteps samples each route. 64 is smooth at globe zoom without bloating
	// the payload; a 40-route globe is then ~2600 coordinate pairs.
	arcSteps = 64
)

// RiskBucket classifies a route. onTimePercent of 0 with no flights means "no
// data", which must not be reported as failure.
func RiskBucket(onTimePercent, avgDelayMinutes float64, flights int) string {
	if flights <= 0 {
		return RiskOK
	}
	if onTimePercent < riskOnTimePercent || avgDelayMinutes >= riskAvgDelayMinutes {
		return RiskRisk
	}
	if onTimePercent < watchOnTimePercent || avgDelayMinutes >= watchAvgDelayMinutes {
		return RiskWatch
	}
	return RiskOK
}

// Feature is a GeoJSON feature with untyped properties.
type Feature struct {
	Type       string         `json:"type"`
	Geometry   Geometry       `json:"geometry"`
	Properties map[string]any `json:"properties"`
}

type Geometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

func emptyCollection() FeatureCollection {
	return FeatureCollection{Type: "FeatureCollection", Features: []Feature{}}
}

// Unresolved reports codes that had no known position, so the UI can say how
// much of the picture is missing instead of silently showing less.
type Unresolved struct {
	Count int      `json:"count"`
	Codes []string `json:"codes"`
}

type Filters struct {
	Airline    string `json:"airline"`
	Airport    string `json:"airport"`
	Risk       string `json:"risk"`
	WindowDays int    `json:"window_days"`
}

// Stats mirrors the numbers rendered in the rail so the canvas and the HTML can
// never disagree.
type Stats struct {
	Routes         int     `json:"routes"`
	Hubs           int     `json:"hubs"`
	Flights        int     `json:"flights"`
	WatchedFlights int     `json:"watched_flights"`
	AtRisk         int     `json:"at_risk"`
	OnTimePercent  float64 `json:"on_time_percent"`
	AvgDelay       float64 `json:"avg_delay_minutes"`
	SampleSize     int     `json:"sample_size"`
	HasAnalytics   bool    `json:"has_analytics"`
	HasOperations  bool    `json:"has_operations"`
	HasTrust       bool    `json:"has_trust"`
}

// Envelope is the payload the globe page bootstraps from and refetches.
//
// Routes, hubs and flights are separate collections so updating live flights
// never touches route geometry.
type Envelope struct {
	GeneratedAt time.Time         `json:"generated_at"`
	WindowDays  int               `json:"window_days"`
	Filters     Filters           `json:"filters"`
	UpstreamMS  int64             `json:"upstream_ms"`
	Routes      FeatureCollection `json:"routes"`
	Hubs        FeatureCollection `json:"hubs"`
	Flights     FeatureCollection `json:"flights"`
	Unresolved  Unresolved        `json:"unresolved"`
	Stats       Stats             `json:"stats"`
}

// CoordLookup resolves an IATA code to a position.
type CoordLookup func(iata string) (lat, lon float64, ok bool)

func pointsToCoords(points []Point) [][]float64 {
	out := make([][]float64, 0, len(points))
	for _, p := range points {
		out = append(out, []float64{round6(p.Lon), round6(p.Lat)})
	}
	return out
}

func round6(v float64) float64 {
	const factor = 1e6
	return float64(int64(v*factor+copysign(0.5, v))) / factor
}

func copysign(v, sign float64) float64 {
	if sign < 0 {
		return -v
	}
	return v
}

// RouteFeature builds one arc. Geometry is LineString, or MultiLineString when
// the route crosses the antimeridian.
func RouteFeature(from, to Point, properties map[string]any) Feature {
	parts := Arc(from, to, arcSteps)

	geometry := Geometry{}
	if len(parts) == 1 {
		geometry.Type = "LineString"
		geometry.Coordinates = pointsToCoords(parts[0])
	} else {
		multi := make([][][]float64, 0, len(parts))
		for _, part := range parts {
			multi = append(multi, pointsToCoords(part))
		}
		geometry.Type = "MultiLineString"
		geometry.Coordinates = multi
	}

	return Feature{Type: "Feature", Geometry: geometry, Properties: properties}
}

// BuildRoutes turns analytics route stats into arc features, resolving both
// endpoints. Routes with an unknown endpoint are dropped and reported.
func BuildRoutes(routes []apiclient.RouteStat, lookup CoordLookup, riskFilter string) (FeatureCollection, []string) {
	collection := emptyCollection()
	unresolved := make([]string, 0)
	seen := map[string]struct{}{}

	if lookup == nil {
		return collection, unresolved
	}

	for _, route := range routes {
		dep := normalizeIATA(route.DepartureIATA)
		arr := normalizeIATA(route.ArrivalIATA)
		if dep == "" || arr == "" || dep == arr {
			continue
		}

		depLat, depLon, depOK := lookup(dep)
		arrLat, arrLon, arrOK := lookup(arr)
		if !depOK {
			unresolved = appendUnique(unresolved, seen, dep)
		}
		if !arrOK {
			unresolved = appendUnique(unresolved, seen, arr)
		}
		if !depOK || !arrOK {
			continue
		}

		risk := RiskBucket(route.OnTimePercent, route.AvgDelayMinutes, route.Flights)
		if riskFilter != "" && risk != riskFilter {
			continue
		}

		collection.Features = append(collection.Features, RouteFeature(
			Point{Lon: depLon, Lat: depLat},
			Point{Lon: arrLon, Lat: arrLat},
			map[string]any{
				"id":          dep + "-" + arr,
				"dep":         dep,
				"arr":         arr,
				"flights":     route.Flights,
				"avg_delay":   route.AvgDelayMinutes,
				"on_time_pct": route.OnTimePercent,
				"risk":        risk,
			},
		))
	}

	return collection, unresolved
}

// BuildHubs turns airport stats into point features sized by traffic.
func BuildHubs(airports []apiclient.AirportStat, lookup CoordLookup) (FeatureCollection, []string) {
	collection := emptyCollection()
	unresolved := make([]string, 0)
	seen := map[string]struct{}{}

	if lookup == nil {
		return collection, unresolved
	}

	for _, airport := range airports {
		iata := normalizeIATA(airport.IATA)
		if iata == "" {
			continue
		}
		lat, lon, ok := lookup(iata)
		if !ok {
			unresolved = appendUnique(unresolved, seen, iata)
			continue
		}

		traffic := airport.Departures + airport.Arrivals
		collection.Features = append(collection.Features, Feature{
			Type: "Feature",
			Geometry: Geometry{
				Type:        "Point",
				Coordinates: []float64{round6(lon), round6(lat)},
			},
			Properties: map[string]any{
				"iata":       iata,
				"departures": airport.Departures,
				"arrivals":   airport.Arrivals,
				"traffic":    traffic,
				"avg_delay":  airport.AvgDelayMinutes,
				"risk":       RiskBucket(100, airport.AvgDelayMinutes, traffic),
			},
		})
	}

	return collection, unresolved
}

// BuildTrustHubs plots the airport dimension of a trust breakdown.
//
// Breakdowns below the low-sample threshold still render, so the map is not
// misleadingly sparse, but they carry low_sample and omit the ratios entirely
// rather than hiding them in CSS.
func BuildTrustHubs(breakdowns []apiclient.TrustBreakdown, lookup CoordLookup, lowSampleThreshold int) (FeatureCollection, []string) {
	collection := emptyCollection()
	unresolved := make([]string, 0)
	seen := map[string]struct{}{}

	if lookup == nil {
		return collection, unresolved
	}

	for _, breakdown := range breakdowns {
		if !strings.EqualFold(breakdown.Dimension, "airport") {
			continue
		}
		iata := normalizeIATA(breakdown.Value)
		if iata == "" {
			continue
		}
		lat, lon, ok := lookup(iata)
		if !ok {
			unresolved = appendUnique(unresolved, seen, iata)
			continue
		}

		properties := map[string]any{
			"iata":      iata,
			"evaluated": breakdown.Evaluated,
			"traffic":   breakdown.Evaluated,
		}
		if breakdown.Evaluated < lowSampleThreshold {
			properties["low_sample"] = true
		} else {
			properties["precision"] = breakdown.Precision
			properties["false_positive_rate"] = breakdown.FalsePositiveRate
		}

		collection.Features = append(collection.Features, Feature{
			Type:       "Feature",
			Geometry:   Geometry{Type: "Point", Coordinates: []float64{round6(lon), round6(lat)}},
			Properties: properties,
		})
	}

	return collection, unresolved
}

// ParseRoute splits a "LIS → LHR" or "LIS-LHR" label into its endpoints.
func ParseRoute(route string) (string, string, bool) {
	route = strings.TrimSpace(route)
	if route == "" {
		return "", "", false
	}
	for _, sep := range []string{"→", "->", "—", "–", "-", "/"} {
		if idx := strings.Index(route, sep); idx > 0 {
			dep := normalizeIATA(route[:idx])
			arr := normalizeIATA(route[idx+len(sep):])
			if dep != "" && arr != "" {
				return dep, arr, true
			}
		}
	}
	return "", "", false
}

func normalizeIATA(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func appendUnique(list []string, seen map[string]struct{}, value string) []string {
	if value == "" {
		return list
	}
	if _, dup := seen[value]; dup {
		return list
	}
	seen[value] = struct{}{}
	return append(list, value)
}

// MergeUnresolved combines unresolved code lists, deduplicated and sorted so the
// footnote is stable between renders.
func MergeUnresolved(lists ...[]string) Unresolved {
	seen := map[string]struct{}{}
	codes := make([]string, 0)
	for _, list := range lists {
		for _, code := range list {
			if code == "" {
				continue
			}
			if _, dup := seen[code]; dup {
				continue
			}
			seen[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	return Unresolved{Count: len(codes), Codes: codes}
}

// FleetMarkerPoints converts resolved fleet markers into flight point features.
func FleetMarkerPoints(markers []models.FleetMarker) FeatureCollection {
	collection := emptyCollection()
	for _, marker := range markers {
		collection.Features = append(collection.Features, Feature{
			Type:     "Feature",
			Geometry: Geometry{Type: "Point", Coordinates: []float64{round6(marker.Lon), round6(marker.Lat)}},
			Properties: map[string]any{
				"id":       marker.Flight,
				"callsign": Callsign(marker.Flight),
				"iata":     marker.IATA,
			},
		})
	}
	return collection
}
