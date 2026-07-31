import { reducedMotion } from '../../core/env.js'
import { arcFeature } from '../great-circle.js'

// Great-circle routes between hub airports for the marketing globe.
// TEMPORARY: P4 replaces these with real aggregate route data from
// /globe/public.json.
const globeRoutes = [
  { from: [-9.13, 38.77], to: [-73.78, 40.64] }, // LIS -> JFK
  { from: [-0.45, 51.47], to: [55.36, 25.25] }, // LHR -> DXB
  { from: [103.99, 1.36], to: [151.18, -33.95] }, // SIN -> SYD
  { from: [-46.47, -23.43], to: [2.55, 49.01] }, // GRU -> CDG
  { from: [139.78, 35.55], to: [-122.38, 37.62] }, // HND -> SFO
  { from: [28.24, -26.14], to: [72.87, 19.09] }, // JNB -> BOM
]

function globeArcFeatures() {
  return {
    type: 'FeatureCollection',
    features: globeRoutes.map(({ from, to }) => arcFeature(from, to)),
  }
}

export function globeLayers(map) {
  map.setProjection({ type: 'globe' })
  map.addSource('globe-arcs', { type: 'geojson', data: globeArcFeatures() })
  map.addLayer({
    id: 'globe-arcs-line',
    type: 'line',
    source: 'globe-arcs',
    paint: {
      'line-color': '#60a5fa',
      'line-width': 1.6,
      'line-opacity': 0.85,
    },
  })
  map.addSource('globe-hubs', {
    type: 'geojson',
    data: {
      type: 'FeatureCollection',
      features: globeRoutes.flatMap(({ from, to }) => [from, to]).map((coordinates) => ({
        type: 'Feature',
        geometry: { type: 'Point', coordinates },
        properties: {},
      })),
    },
  })
  map.addLayer({
    id: 'globe-hubs-dot',
    type: 'circle',
    source: 'globe-hubs',
    paint: {
      'circle-radius': 2.5,
      'circle-color': '#93c5fd',
      'circle-opacity': 0.9,
    },
  })
}

// Slow idle rotation; pauses on interaction, disabled for reduced motion.
export function spinGlobe(map, element) {
  if (reducedMotion()) return
  let userInteracting = false
  const spin = () => {
    if (userInteracting || map.getZoom() > 3) return
    const center = map.getCenter()
    center.lng += 0.4
    map.easeTo({ center, duration: 1000, easing: (n) => n })
  }
  map.on('mousedown', () => { userInteracting = true })
  map.on('dragstart', () => { userInteracting = true })
  map.on('mouseup', () => { userInteracting = false })
  map.on('touchend', () => { userInteracting = false })
  map.on('moveend', spin)
  const observer = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) spin()
      else map.stop()
    })
  }, { threshold: 0.1 })
  observer.observe(element)
  spin()
}
