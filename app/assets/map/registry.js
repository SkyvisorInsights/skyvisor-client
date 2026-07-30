import maplibregl from 'maplibre-gl'
import { flightLayers } from './modes/flight.js'
import { fleetLayers } from './modes/fleet.js'
import { globeLayers, spinGlobe } from './modes/globe.js'

// Collects map containers within a root, including the root itself. htmx
// out-of-band swaps (TrackMapOOB) replace the map element directly, so the
// swapped node *is* the container and a plain querySelectorAll would miss it.
function mapElements(root) {
  const found = []
  if (root.nodeType === 1 && typeof root.matches === 'function' && root.matches('[data-skyvisor-map]')) {
    found.push(root)
  }
  if (typeof root.querySelectorAll === 'function') {
    root.querySelectorAll('[data-skyvisor-map]').forEach((element) => found.push(element))
  }
  return found
}

export function hasMap(root) {
  return mapElements(root).length > 0
}

export function initMaps(root = document) {
  mapElements(root).forEach((element) => {
    const mode = element.dataset.mapMode || 'world'
    const needsReinit = element.dataset.mapReady === 'true' && mode === 'flight'
    if (element.dataset.mapReady === 'true' && !needsReinit) return

    if (needsReinit && element._skyvisorMap) {
      element._skyvisorMap.remove()
      element._skyvisorMap = null
      element.dataset.mapReady = 'false'
    }
    if (element.dataset.mapReady === 'true') return
    element.dataset.mapReady = 'true'

    const center = [Number(element.dataset.longitude || 0), Number(element.dataset.latitude || 28)]
    const map = new maplibregl.Map({
      container: element,
      style: element.dataset.styleUrl || 'https://demotiles.maplibre.org/style.json',
      center,
      zoom: Number(element.dataset.zoom || 1.25),
      attributionControl: false,
      cooperativeGestures: true,
    })
    element._skyvisorMap = map
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'bottom-right')
    map.addControl(new maplibregl.AttributionControl({ compact: true }))

    map.on('load', () => {
      if (mode === 'flight') flightLayers(map, element)
      if (mode === 'fleet') fleetLayers(map, element)
      if (mode === 'globe') {
        globeLayers(map)
        spinGlobe(map, element)
      }
    })
  })
}

// Releases the MapLibre instance (and its WebGL context) for a detached node.
export function teardownMap(element) {
  const map = element && element._skyvisorMap
  if (!map) return
  map.remove()
  element._skyvisorMap = null
  element.dataset.mapReady = 'false'
}
