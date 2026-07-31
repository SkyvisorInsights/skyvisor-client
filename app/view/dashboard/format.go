package dashboard

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
)

func count(value int) string { return strconv.Itoa(value) }

func riskHint(value int) string {
	if value == 0 {
		return "No material signals"
	}
	return "Review priority queue"
}

func freshnessLabel(status string) string {
	switch status {
	case "fresh":
		return "Live data current"
	case "stale":
		return "Some data is stale"
	default:
		return "Freshness unknown"
	}
}

func freshnessDot(status string) string {
	switch status {
	case "fresh":
		return "bg-emerald-500 shadow-[0_0_0_4px_rgba(16,185,129,0.12)]"
	case "stale":
		return "bg-amber-500 shadow-[0_0_0_4px_rgba(245,158,11,0.12)]"
	default:
		return "bg-muted-foreground"
	}
}

func freshnessBadge(status string) string {
	switch status {
	case "fresh":
		return "border-emerald-500/25 bg-emerald-500/8 text-emerald-700 dark:text-emerald-400"
	case "stale":
		return "border-amber-500/30 bg-amber-500/8 text-amber-700 dark:text-amber-400"
	default:
		return "border-border bg-muted/50 text-muted-foreground"
	}
}

func severityDot(severity string) string {
	switch severity {
	case "critical":
		return "bg-destructive shadow-[0_0_0_4px_rgba(239,68,68,0.12)]"
	case "high":
		return "bg-amber-500"
	case "medium":
		return "bg-primary"
	default:
		return "bg-muted-foreground"
	}
}

func severityRail(severity string) string {
	switch severity {
	case "critical":
		return "border-l-destructive"
	case "high":
		return "border-l-amber-500"
	case "medium":
		return "border-l-primary"
	default:
		return "border-l-border"
	}
}

func severityBadge(severity string) string {
	switch severity {
	case "critical":
		return "bg-destructive/10 text-destructive"
	case "high":
		return "bg-amber-500/10 text-amber-700 dark:text-amber-400"
	case "medium":
		return "bg-primary/10 text-primary"
	default:
		return "bg-muted text-muted-foreground"
	}
}

func riskBadge(risk string) string {
	if risk == "miss_likely" {
		return "bg-destructive/10 text-destructive"
	}
	if risk == "tight" {
		return "bg-amber-500/10 text-amber-700 dark:text-amber-400"
	}
	return "bg-muted text-muted-foreground"
}

func riskLabel(risk string) string { return strings.ReplaceAll(risk, "_", " ") }

func providerLabel(provider string) string {
	if strings.TrimSpace(provider) == "" {
		return "Not available"
	}
	return provider
}
func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func timeLabel(value time.Time) string {
	if value.IsZero() {
		return "Not observed"
	}
	return value.Local().Format("15:04:05")
}

func relativeTime(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	d := time.Since(value)
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return value.Local().Format("15:04")
}

func secondsLabel(value int) string {
	if value <= 0 {
		return "Not set"
	}
	if value%60 == 0 {
		return fmt.Sprintf("%d min", value/60)
	}
	return fmt.Sprintf("%d sec", value)
}

func delayLabel(w apiclient.OperationsWatch) string {
	value := 0
	if w.DepartureDelayMinutes != nil && *w.DepartureDelayMinutes > value {
		value = *w.DepartureDelayMinutes
	}
	if w.ArrivalDelayMinutes != nil && *w.ArrivalDelayMinutes > value {
		value = *w.ArrivalDelayMinutes
	}
	if value == 0 {
		return "—"
	}
	return fmt.Sprintf("+%dm", value)
}

func gatesLabel(w apiclient.OperationsWatch) string {
	if w.DepartureGate == "" && w.ArrivalGate == "" {
		return "—"
	}
	return valueOr(w.DepartureGate, "?") + " → " + valueOr(w.ArrivalGate, "?")
}

func fleetOriginIATA(w apiclient.OperationsWatch) string {
	return parseRouteOrigin(w.Route)
}

// watchMarkersJSON serialises pre-resolved fleet markers for the map.
// Coordinates come from the handler, not from a table in the browser.
func watchMarkersJSON(markers []models.FleetMarker) string {
	if len(markers) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(markers))
	for _, marker := range markers {
		parts = append(parts, fmt.Sprintf(
			`{"iata":%q,"flight":%q,"lon":%s,"lat":%s}`,
			marker.IATA, marker.Flight,
			strconv.FormatFloat(marker.Lon, 'f', -1, 64),
			strconv.FormatFloat(marker.Lat, 'f', -1, 64),
		))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func fleetMarkerFor(markers []models.FleetMarker, flightNumber string) (models.FleetMarker, bool) {
	for _, marker := range markers {
		if marker.Flight == flightNumber {
			return marker, true
		}
	}
	return models.FleetMarker{}, false
}

// fleetMarkerLon/Lat return "" when a watch has no resolvable origin, so the
// filmstrip button simply does not fly the map rather than flying it to 0,0.
func fleetMarkerLon(markers []models.FleetMarker, flightNumber string) string {
	marker, ok := fleetMarkerFor(markers, flightNumber)
	if !ok {
		return ""
	}
	return strconv.FormatFloat(marker.Lon, 'f', -1, 64)
}

func fleetMarkerLat(markers []models.FleetMarker, flightNumber string) string {
	marker, ok := fleetMarkerFor(markers, flightNumber)
	if !ok {
		return ""
	}
	return strconv.FormatFloat(marker.Lat, 'f', -1, 64)
}

// FleetMarkersFrom resolves each watch's origin airport to a position.
// Watches whose origin is unknown are dropped rather than plotted at 0,0.
func FleetMarkersFrom(watches []apiclient.OperationsWatch, lookup func(string) (float64, float64, bool)) []models.FleetMarker {
	if len(watches) == 0 || lookup == nil {
		return nil
	}
	markers := make([]models.FleetMarker, 0, len(watches))
	for _, watch := range watches {
		iata := parseRouteOrigin(watch.Route)
		if iata == "" {
			continue
		}
		lat, lon, ok := lookup(iata)
		if !ok {
			continue
		}
		markers = append(markers, models.FleetMarker{
			IATA:   iata,
			Flight: watch.FlightNumber,
			Lat:    lat,
			Lon:    lon,
		})
	}
	return markers
}

func parseRouteOrigin(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return ""
	}
	if idx := strings.Index(route, "→"); idx > 0 {
		return strings.TrimSpace(route[:idx])
	}
	if idx := strings.Index(route, "->"); idx > 0 {
		return strings.TrimSpace(route[:idx])
	}
	parts := strings.Fields(route)
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return ""
}
