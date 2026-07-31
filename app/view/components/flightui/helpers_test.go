package flightui

import (
	"testing"

	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
)

// testLookup stands in for the geo resolver: it knows two airports and nothing
// else, so the unresolved path is exercised too.
func testLookup(iata string) (float64, float64, bool) {
	switch iata {
	case "LIS":
		return 38.7813, -9.1359, true
	case "JFK":
		return 40.6413, -73.7781, true
	default:
		return 0, 0, false
	}
}

func flightWithRoute(depIATA, arrIATA string) *models.LiveFlights {
	flight := &models.LiveFlights{}
	flight.Departure.Iata = depIATA
	flight.Arrival.Iata = arrIATA
	return flight
}

func TestEnrichFlightCoordsWithFillsMissingCoordinates(t *testing.T) {
	t.Parallel()
	flight := flightWithRoute("LIS", "JFK")

	EnrichFlightCoordsWith(flight, testLookup)

	if flight.DepartureLatitude != "38.7813" || flight.DepartureLongitude != "-9.1359" {
		t.Fatalf("departure = (%q, %q), want (38.7813, -9.1359)", flight.DepartureLatitude, flight.DepartureLongitude)
	}
	if flight.ArrivalLatitude != "40.6413" || flight.ArrivalLongitude != "-73.7781" {
		t.Fatalf("arrival = (%q, %q), want (40.6413, -73.7781)", flight.ArrivalLatitude, flight.ArrivalLongitude)
	}
}

func TestEnrichFlightCoordsWithDoesNotOverwriteExisting(t *testing.T) {
	t.Parallel()
	flight := flightWithRoute("LIS", "JFK")
	flight.DepartureLatitude = "1.5"
	flight.DepartureLongitude = "2.5"

	EnrichFlightCoordsWith(flight, testLookup)

	if flight.DepartureLatitude != "1.5" || flight.DepartureLongitude != "2.5" {
		t.Fatalf("existing departure coordinates were overwritten: (%q, %q)", flight.DepartureLatitude, flight.DepartureLongitude)
	}
}

// An unknown airport must stay blank. Filling in zeroes would plot the route
// through the Gulf of Guinea and read as real data.
func TestEnrichFlightCoordsWithLeavesUnknownAirportsBlank(t *testing.T) {
	t.Parallel()
	flight := flightWithRoute("ZZZ", "QQQ")

	EnrichFlightCoordsWith(flight, testLookup)

	if flight.DepartureLatitude != "" || flight.DepartureLongitude != "" {
		t.Fatalf("unknown departure got coordinates (%q, %q)", flight.DepartureLatitude, flight.DepartureLongitude)
	}
	if flight.ArrivalLatitude != "" || flight.ArrivalLongitude != "" {
		t.Fatalf("unknown arrival got coordinates (%q, %q)", flight.ArrivalLatitude, flight.ArrivalLongitude)
	}
}

func TestEnrichFlightCoordsWithHandlesNilInputs(t *testing.T) {
	t.Parallel()
	EnrichFlightCoordsWith(nil, testLookup) // must not panic

	flight := flightWithRoute("LIS", "JFK")
	EnrichFlightCoordsWith(flight, nil)
	if flight.DepartureLatitude != "" {
		t.Fatalf("nil lookup should leave coordinates untouched, got %q", flight.DepartureLatitude)
	}
}

// MapAttrsFromFlight must not resolve anything itself; the handler layer owns
// that so the view package stays free of I/O.
func TestMapAttrsFromFlightIsPure(t *testing.T) {
	t.Parallel()
	flight := flightWithRoute("LIS", "JFK")

	attrs := MapAttrsFromFlight("TP1363", flight)

	if attrs.HasRoute {
		t.Fatal("MapAttrsFromFlight resolved coordinates on its own; it must read only what the flight carries")
	}
	if attrs.FlightNumber != "TP1363" {
		t.Fatalf("FlightNumber = %q", attrs.FlightNumber)
	}

	EnrichFlightCoordsWith(flight, testLookup)
	attrs = MapAttrsFromFlight("TP1363", flight)
	if !attrs.HasRoute {
		t.Fatal("HasRoute = false after enrichment")
	}
	if attrs.DepLat != 38.7813 || attrs.ArrLon != -73.7781 {
		t.Fatalf("attrs = %+v", attrs)
	}
}

func TestMapAttrsFromFlightNilFlight(t *testing.T) {
	t.Parallel()
	attrs := MapAttrsFromFlight("tp1363 ", nil)
	if attrs.FlightNumber != "TP1363" {
		t.Fatalf("FlightNumber = %q, want normalized TP1363", attrs.FlightNumber)
	}
	if attrs.HasRoute || attrs.HasLive {
		t.Fatalf("nil flight produced route/live flags: %+v", attrs)
	}
}
