import maplibregl from 'maplibre-gl'
import { flightLayers } from './modes/flight.js'
import { fleetLayers } from './modes/fleet.js'
import { globeLayers, readBootstrap, renderLabels, setGlobeData, spinGlobe } from './modes/globe.js'
import { resolveMapPalette } from './palette.js'
import { buildStyle, repaint, skySpec } from './style.js'
import { hasWebGL, isLowPowerDevice } from './webgl.js'

// Every live map, so a theme change can repaint them all.
const active = new Set()

function applyPalette() {
  const palette = resolveMapPalette()
  active.forEach((map) => repaint(map, palette))
}

// Themes are announced by core/theme.js. Registered once at module load.
window.addEventListener('skyvisor:theme', applyPalette)

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

// The server-rendered SVG globe stays visible until WebGL has actually drawn
// something. Swapping earlier would flash an empty canvas over a working
// picture; not swapping at all would stack two globes.
function fallbackFor(element) {
  const selector = element.dataset.mapFallback
  return selector ? document.querySelector(selector) : null
}

function revealCanvas(element) {
  element.dataset.mapRendered = 'true'
  const fallback = fallbackFor(element)
  if (fallback) fallback.setAttribute('hidden', '')
}

export function initMaps(root = document) {
  mapElements(root).forEach((element) => {
    const mode = element.dataset.mapMode || 'world'
    const needsReinit = element.dataset.mapReady === 'true' && mode === 'flight'
    if (element.dataset.mapReady === 'true' && !needsReinit) return

    if (needsReinit && element._skyvisorMap) {
      teardownMap(element)
    }
    if (element.dataset.mapReady === 'true') return

    // Without WebGL there is nothing to upgrade to, so leave the server-rendered
    // fallback in place rather than replacing it with an empty container.
    if (!hasWebGL()) {
      element.dataset.mapUnsupported = 'true'
      return
    }
    element.dataset.mapReady = 'true'

    const center = [Number(element.dataset.longitude || 0), Number(element.dataset.latitude || 28)]
    const palette = resolveMapPalette()
    const lowPower = isLowPowerDevice()

    let map
    try {
      map = new maplibregl.Map({
        container: element,
        // Self-hosted Natural Earth basemap. data-style-url still overrides, so
        // a page can point at a different style without touching this code.
        style: element.dataset.styleUrl || buildStyle(palette),
        center,
        zoom: Number(element.dataset.zoom || 1.25),
        attributionControl: false,
        antialias: false,
        maxPitch: 0,
        fadeDuration: 0,
        // Requiring Ctrl to zoom makes sense for a map embedded in a scrolling
        // page, and is hostile on a full-viewport globe the user came to zoom.
        cooperativeGestures: element.dataset.mapGestures !== 'direct',
      })
    } catch (error) {
      console.error('[skyvisor] could not create a map', error)
      element.dataset.mapReady = 'false'
      element.dataset.mapUnsupported = 'true'
      return
    }

    element._skyvisorMap = map
    active.add(map)
    map.on('remove', () => active.delete(map))
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'bottom-right')
    map.addControl(new maplibregl.AttributionControl({ compact: true }))

    map.on('error', (event) => {
      console.error('[skyvisor] map error', event && event.error)
    })

    map.on('load', () => {
      if (typeof map.setSky === 'function') {
        try {
          map.setSky(skySpec(palette))
        } catch {
          // Decorative only.
        }
      }

      if (mode === 'flight') flightLayers(map, element)
      if (mode === 'fleet') fleetLayers(map, element)
      if (mode === 'globe') {
        const data = globeLayers(map, palette, element)

        // The bloom layer is the most expensive thing on the globe and the
        // least load-bearing, so it is the first thing to go on weak hardware.
        if (lowPower && map.getLayer('globe-arcs-glow')) map.removeLayer('globe-arcs-glow')

        element._skyvisorLabels = renderLabels(maplibregl, map, data)
        element._skyvisorSpin = spinGlobe(map, element)
      }
    })

    // idle fires after the first complete render, which is the honest moment to
    // hand over from the server-rendered globe.
    map.once('idle', () => revealCanvas(element))
  })
}

// Applies a fresh envelope to an existing globe without tearing it down.
export function refreshGlobe(element, envelope) {
  const map = element && element._skyvisorMap
  if (!map || !map.isStyleLoaded()) return
  const data = setGlobeData(map, envelope ?? readBootstrap(element))
  if (element._skyvisorLabels) element._skyvisorLabels()
  element._skyvisorLabels = renderLabels(maplibregl, map, data)
}

// Releases the MapLibre instance (and its WebGL context) for a detached node.
export function teardownMap(element) {
  const map = element && element._skyvisorMap
  if (!map) return
  if (element._skyvisorSpin) element._skyvisorSpin()
  if (element._skyvisorLabels) element._skyvisorLabels()
  element._skyvisorSpin = null
  element._skyvisorLabels = null
  active.delete(map)
  map.remove()
  element._skyvisorMap = null
  element.dataset.mapReady = 'false'
  element.dataset.mapRendered = 'false'
}
