import { reducedMotion } from '../../core/env.js'

// Coordinate readout and minimap tracking.
//
// Both are cheap DOM writes driven by map movement, coalesced to one write per
// animation frame — pointer events fire far faster than the screen refreshes,
// and writing on every one is wasted work.

const MINI_WIDTH = 200
const MINI_HEIGHT = 100

function format(value) {
  return value.toFixed(6)
}

// Longitude is unbounded as the globe spins; normalise so the readout shows a
// real-world coordinate rather than 543 degrees east.
function normalizeLon(lon) {
  let value = ((lon + 180) % 360 + 360) % 360 - 180
  if (Object.is(value, -180)) value = 180
  return value
}

export function initGlobeReadout(root = document) {
  const output = (root.querySelector?.('[data-globe-coords]')) || document.querySelector('[data-globe-coords]')
  const marker = document.querySelector('[data-globe-minimap-marker]')
  const canvas = document.querySelector('[data-globe-canvas]')
  const map = canvas && canvas._skyvisorMap
  if (!map || (!output && !marker)) return () => {}
  if (canvas.dataset.readoutReady === 'true') return () => {}
  canvas.dataset.readoutReady = 'true'

  let pending = null
  let lastLon = 0
  let lastLat = 0

  const paint = () => {
    pending = null
    if (output) {
      const [x, y] = output.children
      if (x) x.textContent = `X: ${format(lastLon)}`
      if (y) y.textContent = `Y: ${format(lastLat)}`
    }
    if (marker) {
      marker.setAttribute('cx', (((lastLon + 180) / 360) * MINI_WIDTH).toFixed(1))
      marker.setAttribute('cy', (((90 - lastLat) / 180) * MINI_HEIGHT).toFixed(1))
    }
  }

  const schedule = (lon, lat) => {
    lastLon = normalizeLon(lon)
    lastLat = Math.max(-90, Math.min(90, lat))
    if (pending !== null) return
    pending = requestAnimationFrame(paint)
  }

  // Pointer position when there is a pointer; otherwise the centre of the view,
  // which is the only meaningful "where am I" on a touch device.
  const finePointer = window.matchMedia('(pointer: fine)').matches

  const onMouseMove = (event) => {
    if (!event.lngLat) return
    schedule(event.lngLat.lng, event.lngLat.lat)
  }
  const onMove = () => {
    const center = map.getCenter()
    schedule(center.lng, center.lat)
  }

  if (finePointer) map.on('mousemove', onMouseMove)
  map.on('move', onMove)
  // Seed from the current camera so the readout is never blank.
  onMove()

  return () => {
    if (pending !== null) cancelAnimationFrame(pending)
    if (finePointer) map.off('mousemove', onMouseMove)
    map.off('move', onMove)
    canvas.dataset.readoutReady = 'false'
  }
}

// Projection toggle. The Alpine component owns the pressed state; this listens
// for the resulting event and applies it to the map.
export function initProjectionToggle() {
  const canvas = document.querySelector('[data-globe-canvas]')
  const map = canvas && canvas._skyvisorMap
  if (!map) return () => {}

  const apply = (event) => {
    const value = event.detail === '2d' ? 'mercator' : 'globe'
    try {
      map.setProjection({ type: value })
    } catch (error) {
      console.error('[skyvisor] could not set projection', error)
      return
    }
    // Under reduced motion the morph is suppressed by jumping the camera to
    // where it already is, which cancels MapLibre's animated transition.
    if (reducedMotion()) map.jumpTo({ center: map.getCenter(), zoom: map.getZoom() })
  }

  window.addEventListener('sky:projection', apply)
  return () => window.removeEventListener('sky:projection', apply)
}
