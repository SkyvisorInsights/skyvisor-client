import { reducedMotion } from '../../core/env.js'
import { arcFeature } from '../great-circle.js'

// Fallback routes for the marketing globe, which has no data envelope yet.
// TEMPORARY: P4 replaces these with real aggregate data from /globe/public.json.
const marketingRoutes = [
  { from: [-9.13, 38.77], to: [-73.78, 40.64] }, // LIS -> JFK
  { from: [-0.45, 51.47], to: [55.36, 25.25] }, // LHR -> DXB
  { from: [103.99, 1.36], to: [151.18, -33.95] }, // SIN -> SYD
  { from: [-46.47, -23.43], to: [2.55, 49.01] }, // GRU -> CDG
  { from: [139.78, 35.55], to: [-122.38, 37.62] }, // HND -> SFO
  { from: [28.24, -26.14], to: [72.87, 19.09] }, // JNB -> BOM
]

const EMPTY = { type: 'FeatureCollection', features: [] }

// Labels are DOM markers rather than a symbol layer.
//
// A symbol layer needs an SDF glyph source (~1.5 MB of PBFs to self-host) and
// still could not produce the bordered chip the design calls for. DOM markers
// give the right look with no extra payload — but thousands of them would be a
// guaranteed jank source, so only the most significant few are labelled.
const MAX_HUB_LABELS = 10
const MAX_FLIGHT_LABELS = 12

function marketingCollections() {
  return {
    routes: {
      type: 'FeatureCollection',
      features: marketingRoutes.map(({ from, to }) => arcFeature(from, to, { risk: 'ok', flights: 40 })),
    },
    hubs: {
      type: 'FeatureCollection',
      features: marketingRoutes.flatMap(({ from, to }) => [from, to]).map((coordinates) => ({
        type: 'Feature',
        geometry: { type: 'Point', coordinates },
        properties: { traffic: 20 },
      })),
    },
    flights: EMPTY,
  }
}

// Reads the envelope the server embedded in the page. Returns null when the
// page has no bootstrap, which is how the marketing globe is detected.
export function readBootstrap(element) {
  const selector = element?.dataset?.globeBootstrap
  if (!selector) return null
  const node = document.querySelector(selector)
  if (!node) return null
  try {
    const parsed = JSON.parse(node.textContent || '{}')
    if (!parsed || typeof parsed !== 'object') return null
    return parsed
  } catch (error) {
    console.error('[skyvisor] globe bootstrap is not valid JSON', error)
    return null
  }
}

export function collectionsFrom(envelope) {
  if (!envelope) return marketingCollections()
  return {
    routes: envelope.routes || EMPTY,
    hubs: envelope.hubs || EMPTY,
    flights: envelope.flights || EMPTY,
  }
}

function riskColor(palette) {
  return ['match', ['get', 'risk'], 'risk', palette['arc-risk'], 'watch', palette['arc-watch'], palette['arc-ok']]
}

export function globeLayers(map, palette, element) {
  // Respect the projection the server rendered, so a ?view=2d link does not
  // load as a globe and then visibly flatten.
  map.setProjection({ type: element?.dataset?.globeProjection === '2d' ? 'mercator' : 'globe' })

  const data = collectionsFrom(readBootstrap(element))

  map.addSource('globe-arcs', { type: 'geojson', data: data.routes })
  map.addSource('globe-hubs', { type: 'geojson', data: data.hubs })
  map.addSource('globe-flights', { type: 'geojson', data: data.flights })

  // Soft bloom under the arcs. Dropped on low-core devices by the registry.
  map.addLayer({
    id: 'globe-arcs-glow',
    type: 'line',
    source: 'globe-arcs',
    layout: { 'line-cap': 'round' },
    paint: {
      'line-color': riskColor(palette),
      'line-width': 7,
      'line-blur': 4,
      'line-opacity': 0.16,
    },
  })

  map.addLayer({
    id: 'globe-arcs-line',
    type: 'line',
    source: 'globe-arcs',
    layout: { 'line-cap': 'round' },
    paint: {
      'line-color': riskColor(palette),
      // Busier routes read as heavier lines, clamped so a hub pair does not
      // become a slab.
      'line-width': ['interpolate', ['linear'], ['get', 'flights'], 1, 0.8, 50, 1.6, 400, 3.2],
      'line-opacity': ['interpolate', ['linear'], ['zoom'], 0, 0.55, 4, 0.9],
    },
  })

  map.addLayer({
    id: 'globe-hubs-halo',
    type: 'circle',
    source: 'globe-hubs',
    paint: {
      'circle-radius': ['interpolate', ['linear'], ['sqrt', ['get', 'traffic']], 0, 4, 10, 8.5, 40, 17],
      'circle-color': palette.hub,
      'circle-opacity': 0.14,
    },
  })

  map.addLayer({
    id: 'globe-hubs-dot',
    type: 'circle',
    source: 'globe-hubs',
    paint: {
      // Area, not radius, tracks traffic.
      'circle-radius': ['interpolate', ['linear'], ['sqrt', ['get', 'traffic']], 0, 2.2, 10, 4.5, 40, 9],
      'circle-color': palette.hub,
      'circle-stroke-width': 1,
      'circle-stroke-color': palette.ocean,
      'circle-opacity': 0.95,
    },
  })

  map.addLayer({
    id: 'globe-flights-glow',
    type: 'circle',
    source: 'globe-flights',
    paint: {
      'circle-radius': 6,
      'circle-color': palette.flight,
      'circle-opacity': 0.25,
    },
  })

  // A non-directional puck: FlightLive carries no heading, so rotating an
  // aircraft glyph here would invent a bearing the data does not have.
  map.addLayer({
    id: 'globe-flights-dot',
    type: 'circle',
    source: 'globe-flights',
    paint: {
      'circle-radius': 3,
      'circle-color': palette.flight,
      'circle-stroke-width': 1.5,
      'circle-stroke-color': riskColor(palette),
      'circle-opacity': ['case', ['==', ['get', 'stale'], true], 0.35, 1],
    },
  })

  return data
}

// Replaces the whole dataset in place, used by the htmx refresh path.
export function setGlobeData(map, envelope) {
  const data = collectionsFrom(envelope)
  const apply = (id, collection) => {
    const source = map.getSource(id)
    if (source) source.setData(collection)
  }
  apply('globe-arcs', data.routes)
  apply('globe-hubs', data.hubs)
  apply('globe-flights', data.flights)
  return data
}

function chip(text, kind) {
  const node = document.createElement('span')
  node.className = 'skyvisor-map-chip'
  node.dataset.chipKind = kind
  node.textContent = text
  return node
}

// Labels the most significant features. Returns a disposer so a data refresh
// drops the old markers instead of stacking new ones on top.
export function renderLabels(maplibregl, map, data) {
  const markers = []

  const hubs = [...(data.hubs?.features || [])]
    .filter((feature) => feature.properties?.iata)
    .sort((a, b) => (b.properties.traffic || 0) - (a.properties.traffic || 0))
    .slice(0, MAX_HUB_LABELS)

  for (const feature of hubs) {
    markers.push(
      new maplibregl.Marker({ element: chip(feature.properties.iata, 'hub'), anchor: 'left', offset: [10, 0] })
        .setLngLat(feature.geometry.coordinates)
        .addTo(map),
    )
  }

  const flights = [...(data.flights?.features || [])]
    .sort((a, b) => (b.properties?.delay_minutes || 0) - (a.properties?.delay_minutes || 0))
    .slice(0, MAX_FLIGHT_LABELS)

  for (const feature of flights) {
    const label = feature.properties?.callsign || feature.properties?.id
    if (!label) continue
    markers.push(
      new maplibregl.Marker({ element: chip(label, 'flight'), anchor: 'bottom', offset: [0, -10] })
        .setLngLat(feature.geometry.coordinates)
        .addTo(map),
    )
  }

  return () => markers.forEach((marker) => marker.remove())
}

// Slow idle rotation; pauses on interaction, disabled for reduced motion.
export function spinGlobe(map, element) {
  if (reducedMotion()) return () => {}

  let userInteracting = false
  let stopped = false

  const spin = () => {
    if (stopped || userInteracting || map.getZoom() > 3) return
    const center = map.getCenter()
    center.lng += 0.4
    map.easeTo({ center, duration: 1000, easing: (n) => n })
  }

  const hold = () => { userInteracting = true }
  const release = () => { userInteracting = false }

  map.on('mousedown', hold)
  map.on('dragstart', hold)
  map.on('mouseup', release)
  map.on('touchend', release)
  // Scroll and keyboard users were previously fighting the rotation, because
  // only pointer drags paused it.
  map.on('wheel', hold)
  map.on('zoomstart', hold)
  map.on('zoomend', release)
  map.on('moveend', spin)
  map.getCanvas().addEventListener('keydown', hold)

  const observer = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        stopped = false
        spin()
      } else {
        stopped = true
        map.stop()
      }
    })
  }, { threshold: 0.1 })
  observer.observe(element)

  // A hidden tab should not keep a WebGL render loop and an easeTo running.
  const onVisibility = () => {
    stopped = document.hidden
    if (!stopped) spin()
  }
  document.addEventListener('visibilitychange', onVisibility)

  spin()

  return () => {
    stopped = true
    observer.disconnect()
    document.removeEventListener('visibilitychange', onVisibility)
  }
}
