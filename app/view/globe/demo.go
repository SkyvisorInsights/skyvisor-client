package globe

import (
	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
)

// Demo data for local development.
//
// This exists so the globe can be worked on without a live skyvisor-api
// session. It is gated behind an explicit environment variable, is never used
// when a real session is present, and the page it feeds is required to label
// itself as demo data — an unlabelled globe of invented flights is exactly the
// kind of confidently-wrong dashboard the rest of this package is built to
// avoid.
//
// Coordinates are not invented: the IATA codes below are resolved through the
// same geo resolver as real data, so anything the database does not know is
// dropped and reported like any other unresolvable airport.

var demoRoutes = []apiclient.RouteStat{
	{DepartureIATA: "LIS", ArrivalIATA: "JFK", Flights: 128, OnTimePercent: 91.2, AvgDelayMinutes: 6},
	{DepartureIATA: "LIS", ArrivalIATA: "LHR", Flights: 210, OnTimePercent: 88.4, AvgDelayMinutes: 9},
	{DepartureIATA: "LHR", ArrivalIATA: "DXB", Flights: 96, OnTimePercent: 76.1, AvgDelayMinutes: 18},
	{DepartureIATA: "CDG", ArrivalIATA: "GRU", Flights: 44, OnTimePercent: 58.3, AvgDelayMinutes: 34},
	{DepartureIATA: "FRA", ArrivalIATA: "SIN", Flights: 61, OnTimePercent: 82.0, AvgDelayMinutes: 12},
	{DepartureIATA: "HND", ArrivalIATA: "SFO", Flights: 38, OnTimePercent: 41.5, AvgDelayMinutes: 52},
	{DepartureIATA: "AMS", ArrivalIATA: "MAD", Flights: 154, OnTimePercent: 93.7, AvgDelayMinutes: 4},
	{DepartureIATA: "JFK", ArrivalIATA: "LAX", Flights: 187, OnTimePercent: 79.2, AvgDelayMinutes: 16},
	{DepartureIATA: "DXB", ArrivalIATA: "SYD", Flights: 27, OnTimePercent: 85.1, AvgDelayMinutes: 11},
	{DepartureIATA: "ORD", ArrivalIATA: "FCO", Flights: 33, OnTimePercent: 68.9, AvgDelayMinutes: 24},
	{DepartureIATA: "ATL", ArrivalIATA: "MIA", Flights: 240, OnTimePercent: 95.4, AvgDelayMinutes: 3},
	{DepartureIATA: "IST", ArrivalIATA: "DOH", Flights: 72, OnTimePercent: 87.2, AvgDelayMinutes: 8},
	// Deliberately unresolvable, so the "airports hidden" footnote is exercised.
	{DepartureIATA: "LIS", ArrivalIATA: "ZZZ", Flights: 12, OnTimePercent: 80, AvgDelayMinutes: 10},
}

var demoAirports = []apiclient.AirportStat{
	{IATA: "LIS", Departures: 340, Arrivals: 322, AvgDelayMinutes: 7},
	{IATA: "LHR", Departures: 610, Arrivals: 598, AvgDelayMinutes: 14},
	{IATA: "JFK", Departures: 480, Arrivals: 465, AvgDelayMinutes: 12},
	{IATA: "CDG", Departures: 402, Arrivals: 396, AvgDelayMinutes: 19},
	{IATA: "FRA", Departures: 388, Arrivals: 377, AvgDelayMinutes: 9},
	{IATA: "DXB", Departures: 356, Arrivals: 348, AvgDelayMinutes: 11},
	{IATA: "SIN", Departures: 244, Arrivals: 238, AvgDelayMinutes: 6},
	{IATA: "HND", Departures: 298, Arrivals: 289, AvgDelayMinutes: 31},
	{IATA: "AMS", Departures: 331, Arrivals: 320, AvgDelayMinutes: 8},
	{IATA: "GRU", Departures: 176, Arrivals: 170, AvgDelayMinutes: 22},
	{IATA: "LAX", Departures: 412, Arrivals: 401, AvgDelayMinutes: 15},
	{IATA: "SYD", Departures: 158, Arrivals: 151, AvgDelayMinutes: 5},
	{IATA: "MAD", Departures: 266, Arrivals: 259, AvgDelayMinutes: 10},
	{IATA: "FCO", Departures: 201, Arrivals: 195, AvgDelayMinutes: 17},
	{IATA: "ATL", Departures: 690, Arrivals: 672, AvgDelayMinutes: 4},
	{IATA: "SFO", Departures: 288, Arrivals: 279, AvgDelayMinutes: 13},
}

// demoFlight describes an aircraft partway along a route.
type demoFlight struct {
	Number   string
	Dep      string
	Arr      string
	Progress float64
	Delay    int
	Stale    bool
	Status   string
}

var demoFlights = []demoFlight{
	{Number: "TP1363", Dep: "LIS", Arr: "LHR", Progress: 0.42, Delay: 0, Status: "en_route"},
	{Number: "TP204", Dep: "LIS", Arr: "JFK", Progress: 0.61, Delay: 12, Status: "en_route"},
	{Number: "BA106", Dep: "LHR", Arr: "DXB", Progress: 0.28, Delay: 24, Status: "en_route"},
	{Number: "AF456", Dep: "CDG", Arr: "GRU", Progress: 0.77, Delay: 41, Status: "en_route"},
	{Number: "LH778", Dep: "FRA", Arr: "SIN", Progress: 0.53, Delay: 6, Status: "en_route"},
	{Number: "NH7", Dep: "HND", Arr: "SFO", Progress: 0.35, Delay: 58, Status: "en_route"},
	{Number: "KL1701", Dep: "AMS", Arr: "MAD", Progress: 0.66, Delay: 0, Status: "en_route"},
	{Number: "AA118", Dep: "JFK", Arr: "LAX", Progress: 0.49, Delay: 18, Status: "en_route"},
	// A stale position: last observed long enough ago that it must be shown
	// frozen and dimmed rather than smoothly interpolated onward.
	{Number: "EK412", Dep: "DXB", Arr: "SYD", Progress: 0.19, Delay: 9, Stale: true, Status: "en_route"},
	{Number: "DL84", Dep: "ORD", Arr: "FCO", Progress: 0.71, Delay: 33, Status: "en_route"},
}

var demoAttention = []apiclient.OperationsAttention{
	{ID: "demo:1", Severity: "high", Kind: "delay", Title: "NH7 is delayed 58 minutes", FlightNumber: "NH7", Route: "HND → SFO", Reason: "Inbound aircraft is late out of Tokyo."},
	{ID: "demo:2", Severity: "high", Kind: "connection", Title: "AF456 threatens a connection", FlightNumber: "AF456", Route: "CDG → GRU", Reason: "41 minute delay against a 55 minute layover."},
	{ID: "demo:3", Severity: "medium", Kind: "delay", Title: "DL84 is delayed 33 minutes", FlightNumber: "DL84", Route: "ORD → FCO", Reason: "Air traffic flow restriction over the Alps."},
	{ID: "demo:4", Severity: "medium", Kind: "delay", Title: "BA106 is delayed 24 minutes", FlightNumber: "BA106", Route: "LHR → DXB", Reason: "Late crew connection."},
	{ID: "demo:5", Severity: "low", Kind: "gate", Title: "AA118 changed gate", FlightNumber: "AA118", Route: "JFK → LAX", Reason: "Now departing from gate B14."},
	{ID: "demo:6", Severity: "low", Kind: "status", Title: "TP1363 is on time", FlightNumber: "TP1363", Route: "LIS → LHR", Reason: "No action needed."},
}

// DemoAnalytics returns the stand-in analytics report.
func DemoAnalytics() apiclient.AnalyticsReport {
	return apiclient.AnalyticsReport{
		WindowDays: 7,
		Routes:     demoRoutes,
		Airports:   demoAirports,
		SampleSize: 2410,
		Summary: apiclient.AnalyticsSummary{
			FlightCount:              2410,
			OnTimePercent:            84.6,
			AvgDepartureDelayMinutes: 11.4,
		},
	}
}

func DemoAttention() []apiclient.OperationsAttention { return demoAttention }

// DemoFlightPoints places each demo aircraft along its great circle, using real
// resolved airport coordinates.
func DemoFlightPoints(lookup CoordLookup) FeatureCollection {
	collection := emptyCollection()
	if lookup == nil {
		return collection
	}

	for _, flight := range demoFlights {
		depLat, depLon, depOK := lookup(flight.Dep)
		arrLat, arrLon, arrOK := lookup(flight.Arr)
		if !depOK || !arrOK {
			continue
		}

		position := Interpolate(
			Point{Lon: depLon, Lat: depLat},
			Point{Lon: arrLon, Lat: arrLat},
			flight.Progress,
		)

		risk := RiskOK
		switch {
		case flight.Delay >= 30:
			risk = RiskRisk
		case flight.Delay >= 15:
			risk = RiskWatch
		}

		properties := map[string]any{
			"id":            flight.Number,
			"callsign":      Callsign(flight.Number),
			"number":        flight.Number,
			"dep":           flight.Dep,
			"arr":           flight.Arr,
			"status":        flight.Status,
			"delay_minutes": flight.Delay,
			"risk":          risk,
		}
		if flight.Stale {
			properties["stale"] = true
		}

		collection.Features = append(collection.Features, Feature{
			Type:       "Feature",
			Geometry:   Geometry{Type: "Point", Coordinates: []float64{round6(position.Lon), round6(position.Lat)}},
			Properties: properties,
		})
	}

	return collection
}

// DemoWatchCount is what the header reports as watched, so "N watched · M
// observed" stays consistent with the pucks actually drawn.
func DemoWatchCount() int { return len(demoFlights) }
