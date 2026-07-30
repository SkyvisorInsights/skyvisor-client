import maplibregl from 'maplibre-gl'
import { parseNumber, reducedMotion } from '../../core/env.js'
import { greatCirclePoints, routePointAtProgress } from '../geometry.js'

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
  const route = greatCirclePoints(dep, arr)
  const progress = parseNumber(element.dataset.progress) ?? 42
  const planePos = liveLon != null && liveLat != null ? [liveLon, liveLat] : routePointAtProgress(dep, arr, progress)

  if (!map.getSource('flight-route')) {
    map.addSource('flight-route', {
      type: 'geojson',
      data: { type: 'Feature', geometry: { type: 'LineString', coordinates: route }, properties: {} },
    })
    map.addLayer({
      id: 'flight-route-line',
      type: 'line',
      source: 'flight-route',
      paint: { 'line-color': '#60a5fa', 'line-width': 2.5, 'line-opacity': 0.85 },
    })
    map.addSource('flight-plane', {
      type: 'geojson',
      data: { type: 'Feature', geometry: { type: 'Point', coordinates: planePos }, properties: {} },
    })
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
    map.getSource('flight-route').setData({ type: 'Feature', geometry: { type: 'LineString', coordinates: route }, properties: {} })
    map.getSource('flight-plane').setData({ type: 'Feature', geometry: { type: 'Point', coordinates: planePos }, properties: {} })
  }

  const bounds = route.reduce((acc, coord) => acc.extend(coord), new maplibregl.LngLatBounds(route[0], route[0]))
  map.fitBounds(bounds, { padding: 72, maxZoom: 6.5, duration: reducedMotion() ? 0 : 900 })
}
