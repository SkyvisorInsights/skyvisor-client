// Hardcoded airport coordinates.
//
// TEMPORARY. This is one of three divergent copies of the same table (the
// others are airportCoord() in app/view/components/flightui/helpers.go and
// Self.airports in skyvisor-ios FlightMapView.swift). P1 replaces this with
// server-resolved coordinates from the `airport` table, which has ~10k rows
// instead of these 20, and deletes this file.
export const AIRPORT_COORDS = {
  LIS: [-9.1359, 38.7813],
  LHR: [-0.4543, 51.47],
  JFK: [-73.7781, 40.6413],
  LAX: [-118.4085, 33.9425],
  CDG: [2.5479, 49.0097],
  FRA: [8.5622, 50.0379],
  DXB: [55.3657, 25.2532],
  SIN: [103.9915, 1.3644],
  HND: [139.7798, 35.5494],
  SYD: [151.177, -33.9399],
  GRU: [-46.473, -23.4356],
  ORD: [-87.9048, 41.9786],
  ATL: [-84.4281, 33.6367],
  DFW: [-97.038, 32.8998],
  MIA: [-80.2906, 25.7959],
  AMS: [4.7639, 52.3105],
  MAD: [-3.5676, 40.4983],
  FCO: [12.2389, 41.8003],
  IST: [28.8146, 41.2753],
  DOH: [51.608, 25.2731],
}
