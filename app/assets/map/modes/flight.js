import maplibregl from 'maplibre-gl'
import { parseNumber, reducedMotion } from '../../core/env.js'
import { arcFeature, arcPoints, pointAtProgress } from '../great-circle.js'

export function flightLayers(map, element) {
  const depLon = parseNumber(element.dataset.depLon)
  const depLat = parseNumber(element.dataset.depLat)
  const arrLon = parseNumber(element.dataset.arrLon)
  const arrLat = parseNumber(element.dataset.arrLat)
  const liveLon = parseNumber(element.dataset.liveLon)
  const liveLat = parseNumber(element.dataset.liveLat)
  if (depLon == null || depLat == null || arrLon == null || arrLat == null) return

  const dep = [depLon, depLat]
  const arr = [arrLon, arrLat]
  const progress = parseNumber(element.dataset.progress) ?? 42
  const routeFeature = arcFeature(dep, arr)
  const planePos = liveLon != null && liveLat != null
    ? [liveLon, liveLat]
    : pointAtProgress(dep, arr, progress / 100)
  const planeFeature = { type: 'Feature', geometry: { type: 'Point', coordinates: planePos }, properties: {} }

  if (!map.getSource('flight-route')) {
    map.addSource('flight-route', { type: 'geojson', data: routeFeature })
    map.addLayer({
      id: 'flight-route-line',
      type: 'line',
      source: 'flight-route',
      paint: { 'line-color': '#60a5fa', 'line-width': 2.5, 'line-opacity': 0.85 },
    })
    map.addSource('flight-plane', { type: 'geojson', data: planeFeature })
    map.addLayer({
      id: 'flight-plane-dot',
      type: 'circle',
      source: 'flight-plane',
      paint: {
        'circle-radius': 7,
        'circle-color': '#38bdf8',
        'circle-stroke-width': 2,
        'circle-stroke-color': '#ffffff',
      },
    })
  } else {
    map.getSource('flight-route').setData(routeFeature)
    map.getSource('flight-plane').setData(planeFeature)
  }

  // Bounds are computed from sampled points rather than the arc geometry so
  // antimeridian-split MultiLineStrings do not need special handling here.
  const sampled = arcPoints(dep, arr, 48)
  const bounds = sampled.reduce((acc, coord) => acc.extend(coord), new maplibregl.LngLatBounds(sampled[0], sampled[0]))
  map.fitBounds(bounds, { padding: 72, maxZoom: 6.5, duration: reducedMotion() ? 0 : 900 })
}
