// Fleet markers are fully resolved server-side (see FleetMarkersFrom in
// app/view/dashboard/format.go). Markers without a known origin are dropped
// there, so anything arriving here already has a real position.
export function fleetLayers(map, element) {
  let markers = []
  try {
    markers = JSON.parse(element.dataset.markers || '[]')
  } catch {
    markers = []
  }

  const features = markers
    .filter((marker) => Number.isFinite(marker.lon) && Number.isFinite(marker.lat))
    .map((marker) => ({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [marker.lon, marker.lat] },
      properties: { label: marker.flight || marker.iata },
    }))

  const data = { type: 'FeatureCollection', features }
  if (!map.getSource('fleet-markers')) {
    map.addSource('fleet-markers', { type: 'geojson', data })
    map.addLayer({
      id: 'fleet-markers-dot',
      type: 'circle',
      source: 'fleet-markers',
      paint: {
        'circle-radius': 5,
        'circle-color': '#38bdf8',
        'circle-stroke-width': 1.5,
        'circle-stroke-color': '#ffffff',
      },
    })
  } else {
    map.getSource('fleet-markers').setData(data)
  }
}
