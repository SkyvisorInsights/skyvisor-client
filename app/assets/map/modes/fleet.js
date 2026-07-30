import { AIRPORT_COORDS } from '../coords.js'

export function fleetLayers(map, element) {
  let markers = []
  try {
    markers = JSON.parse(element.dataset.markers || '[]')
  } catch {
    markers = []
  }
  const features = markers
    .map((marker) => {
      const coord = AIRPORT_COORDS[marker.iata]
      if (!coord) return null
      return {
        type: 'Feature',
        geometry: { type: 'Point', coordinates: coord },
        properties: { label: marker.flight || marker.iata },
      }
    })
    .filter(Boolean)

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
