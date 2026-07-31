package models

// AirportCoord is a minimal, hot-path-friendly airport record.
//
// Deliberately narrower than Airport: that struct carries sql.NullString and a
// CustomTime, which make it awkward to hold ~10k of in a long-lived cache.
type AirportCoord struct {
	IATA        string
	ICAO        string
	Lat         float64
	Lon         float64
	Name        string
	CityIATA    string
	CountryISO2 string
	Timezone    string
}

// FleetMarker is a watched flight positioned at its origin airport.
// Coordinates are resolved server-side so the browser never needs its own
// airport table.
type FleetMarker struct {
	IATA   string
	Flight string
	Lat    float64
	Lon    float64
}
