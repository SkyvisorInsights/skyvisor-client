import { subscribe } from '../../core/live.js'
import { reducedMotion } from '../../core/env.js'

// Live flight positions on the globe.
//
// Two sources feed this, deliberately:
//
//   SSE gives low-latency updates, but the hub is per-account and driven by
//   watches, so it will never carry every aircraft on the globe.
//   A 60s reconcile poll replaces the whole set, so anything SSE cannot see
//   still appears, and anything that has gone away disappears.
//
// The UI states both numbers ("N watched · M observed") rather than implying
// the globe is a complete picture of the sky.

const FLIGHT_EVENTS = ['flight.updated', 'flight.delayed', 'flight.cancelled']
const RECONCILE_MS = 60_000
const SMOOTH_MS = 800

function featureFrom(flight) {
  const live = flight && flight.live
  if (!live || typeof live.latitude !== 'number' || typeof live.longitude !== 'number') return null

  const id = flight.number || flight.flight_number
  if (!id) return null

  return {
    type: 'Feature',
    geometry: { type: 'Point', coordinates: [live.longitude, live.latitude] },
    properties: {
      id,
      callsign: String(id).replace(/^([A-Z0-9]{2,3})(\d)/i, '$1-$2').toUpperCase(),
      dep: flight.departure_iata || '',
      arr: flight.arrival_iata || '',
      status: flight.status || '',
      delay_minutes: flight.departure_delay_minutes || 0,
      updated_at: live.updated_at || null,
    },
  }
}

export function initGlobeLive(element, options = {}) {
  const map = element && element._skyvisorMap
  if (!map) return () => {}

  const sourceId = 'globe-flights'
  const index = new Map()
  const animations = new Map()
  let flushHandle = null
  let disposed = false

  // Seed from whatever the server already drew, so an SSE update for one
  // aircraft does not wipe the rest.
  const seed = () => {
    index.clear()
    const source = map.getSource(sourceId)
    const data = source && source._data
    for (const feature of (data && data.features) || []) {
      if (feature.properties?.id) index.set(feature.properties.id, feature)
    }
  }
  seed()

  const flush = () => {
    flushHandle = null
    const source = map.getSource(sourceId)
    if (!source) return
    source.setData({ type: 'FeatureCollection', features: [...index.values()] })
  }

  // Coalesce to one setData per frame: a burst of SSE events must not trigger a
  // GeoJSON re-parse per message.
  const scheduleFlush = () => {
    if (disposed || flushHandle !== null) return
    flushHandle = requestAnimationFrame(flush)
  }

  // Interpolating between two observed positions is legitimate. Extrapolating
  // past the last observation is not — a dead-reckoned aircraft looks most
  // convincing exactly when it is most wrong — so nothing here runs on a timer
  // beyond the two points we were actually given.
  const glide = (id, from, to) => {
    const existing = animations.get(id)
    if (existing) cancelAnimationFrame(existing)

    if (reducedMotion() || !from) {
      scheduleFlush()
      return
    }

    const start = performance.now()
    const step = (now) => {
      const t = Math.min(1, (now - start) / SMOOTH_MS)
      const feature = index.get(id)
      if (!feature) return
      feature.geometry.coordinates = [
        from[0] + (to[0] - from[0]) * t,
        from[1] + (to[1] - from[1]) * t,
      ]
      scheduleFlush()
      if (t < 1) animations.set(id, requestAnimationFrame(step))
      else animations.delete(id)
    }
    animations.set(id, requestAnimationFrame(step))
  }

  const upsert = (feature) => {
    const id = feature.properties.id
    const previous = index.get(id)
    const from = previous ? [...previous.geometry.coordinates] : null
    const to = feature.geometry.coordinates

    // Carry forward server-derived fields the event does not include.
    if (previous) {
      feature.properties = { ...previous.properties, ...feature.properties }
      feature.geometry.coordinates = from
    }
    index.set(id, feature)

    if (previous) glide(id, from, to)
    else scheduleFlush()
  }

  const release = subscribe(FLIGHT_EVENTS, (payload) => {
    const feature = featureFrom(payload && payload.flight)
    if (feature) upsert(feature)
  })

  // Reconcile. Paused while the tab is hidden — polling a globe nobody is
  // looking at spends the user's battery and our rate limit for nothing.
  let timer = null
  const reconcile = async () => {
    if (disposed || document.hidden) return
    try {
      const response = await fetch(options.dataUrl || '/globe/data', {
        credentials: 'same-origin',
        headers: { Accept: 'application/json' },
      })
      if (!response.ok) return
      const envelope = await response.json()
      index.clear()
      for (const feature of envelope.flights?.features || []) {
        if (feature.properties?.id) index.set(feature.properties.id, feature)
      }
      scheduleFlush()
      if (typeof options.onEnvelope === 'function') options.onEnvelope(envelope, response)
    } catch {
      // A failed reconcile leaves the last known positions on screen; the
      // freshness chip is what tells the user they are ageing.
    }
  }

  const startTimer = () => {
    if (timer !== null) return
    timer = window.setInterval(reconcile, RECONCILE_MS)
  }
  const stopTimer = () => {
    if (timer === null) return
    window.clearInterval(timer)
    timer = null
  }

  const onVisibility = () => (document.hidden ? stopTimer() : startTimer())
  document.addEventListener('visibilitychange', onVisibility)
  if (!document.hidden) startTimer()

  return () => {
    disposed = true
    release()
    stopTimer()
    document.removeEventListener('visibilitychange', onVisibility)
    animations.forEach((handle) => cancelAnimationFrame(handle))
    animations.clear()
    if (flushHandle !== null) cancelAnimationFrame(flushHandle)
  }
}
