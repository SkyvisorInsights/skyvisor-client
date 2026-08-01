package globe

import (
	"math"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
)

// AttentionRows converts the operations attention queue into drawer rows,
// resolving each route's origin so focusing a row can move the globe.
//
// Rows with no resolvable position are still listed — the row is the accessible
// equivalent of the canvas, so dropping it would hide information rather than
// just hide a dot. Only the fly-to affordance is withheld.
func AttentionRows(items []apiclient.OperationsAttention, lookup func(string) (float64, float64, bool)) []AttentionRow {
	rows := make([]AttentionRow, 0, len(items))

	for _, item := range items {
		dep, arr, _ := ParseRoute(item.Route)
		row := AttentionRow{
			ID:       item.ID,
			Severity: item.Severity,
			Title:    item.Title,
			Flight:   item.FlightNumber,
			Route:    item.Route,
			Dep:      dep,
			Arr:      arr,
			Reason:   item.Reason,
		}

		if lookup != nil && dep != "" {
			if lat, lon, ok := lookup(dep); ok {
				row.Lat, row.Lon, row.HasPoint = lat, lon, true
			}
		}

		rows = append(rows, row)
	}

	return rows
}

// FleetMarkersFromWatches resolves each watched flight to its origin airport.
// Watches whose origin is unknown are dropped rather than plotted at 0,0.
func FleetMarkersFromWatches(watches []apiclient.OperationsWatch, lookup func(string) (float64, float64, bool)) []models.FleetMarker {
	if len(watches) == 0 || lookup == nil {
		return nil
	}

	markers := make([]models.FleetMarker, 0, len(watches))
	for _, watch := range watches {
		dep, _, ok := ParseRoute(watch.Route)
		if !ok || dep == "" {
			continue
		}
		lat, lon, found := lookup(dep)
		if !found {
			continue
		}
		markers = append(markers, models.FleetMarker{
			IATA:   dep,
			Flight: watch.FlightNumber,
			Lat:    lat,
			Lon:    lon,
		})
	}
	return markers
}

// CountRisk counts features in a given risk bucket.
func CountRisk(collection FeatureCollection, risk string) int {
	count := 0
	for _, feature := range collection.Features {
		if featureRisk(feature) == risk {
			count++
		}
	}
	return count
}

// MeanLongitude finds a longitude that centres the rendered data.
//
// Longitudes are averaged as unit vectors rather than arithmetically, because
// the mean of -179 and 179 is 0 the naive way when the answer is 180.
func MeanLongitude(collection FeatureCollection) (float64, bool) {
	var sumX, sumY float64
	count := 0

	for _, feature := range collection.Features {
		for _, line := range lineStrings(feature.Geometry) {
			for _, pair := range line {
				if len(pair) != 2 {
					continue
				}
				rad := pair[0] * degToRad
				sumX += math.Cos(rad)
				sumY += math.Sin(rad)
				count++
			}
		}
		if lon, _, ok := coordPair(feature.Geometry.Coordinates); ok {
			rad := lon * degToRad
			sumX += math.Cos(rad)
			sumY += math.Sin(rad)
			count++
		}
	}

	if count == 0 || (sumX == 0 && sumY == 0) {
		return 0, false
	}
	return math.Atan2(sumY, sumX) / degToRad, true
}
