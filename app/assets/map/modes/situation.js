// The situation map mode: many independently toggleable layers over the globe.
//
// Deliberately a separate mode rather than a flag on the globe mode. The globe
// hardcodes three sources and runs an idle spin loop; situation has a variable
// number of sources driven by a server-served catalogue, no spin, and
// click-to-inspect.

import { paletteKeyFor, specFor } from './situation-layers.js'

// Severity bands as the API emits them, worst last so the expression reads in
// increasing order of alarm.
const SEVERITIES = ['info', 'advisory', 'watch', 'warning', 'severe']

// situationSourceId and layer ids are derived rather than free-form so the
// toggle, the repaint and the tests all agree on a name without sharing state.
// Dots are not valid in the identifiers MapLibre style expressions reference
// comfortably, so they become dashes.
export function situationSourceId(layerID) {
  return `situation-${String(layerID).replace(/\./g, '-')}`
}

function layerIdsFor(layerID, spec) {
  const base = situationSourceId(layerID)
  if (spec.type === 'fill') return [`${base}-fill`, `${base}-line`]
  return [`${base}-dot`]
}

// severityColor grades a feature by its severity property.
//
// A flat colour would make a volcanic ash advisory look like light turbulence.
// It must stay an expression: repainting with a flat value on theme change
// would silently destroy the grading, which is why repaintSituation exists
// rather than letting the generic repaint() handle these layers.
export function severityColor(palette) {
  const base = palette['situation-hazard'] || '#f0a63a'
  return [
    'match',
    ['get', 'severity'],
    'info', palette['situation-info'] || base,
    'advisory', palette['situation-advisory'] || base,
    'watch', palette['situation-watch'] || base,
    'warning', palette['situation-warning'] || base,
    'severe', palette['situation-severe'] || base,
    base,
  ]
}

// radiusFor scales a circle by a metric when the spec names one, so magnitude
// and fire radiative power read at a glance instead of needing a click.
function radiusFor(spec) {
  if (!spec.radiusMetric) return 4
  return [
    'interpolate',
    ['linear'],
    ['coalesce', ['get', spec.radiusMetric], 0],
    0, 3,
    10, 14,
  ]
}

// createdLayers remembers what this module put on a map and which palette key
// each layer uses.
//
// The static PAINT_BINDINGS list in style.js cannot serve here: the catalogue
// is fetched from the API, so a layer added server-side would never appear in a
// list compiled into this client. Recording the layers as they are created is
// what lets the theme repaint cover them anyway.
const createdLayers = new WeakMap()

const EMPTY = { type: 'FeatureCollection', features: [] }

// situationLayers creates a source and layers for every entitled layer in the
// catalogue, all hidden.
//
// Everything is created once at load and toggled by visibility afterwards.
// addLayer after load re-sorts and re-validates the entire style, which
// visibly hitches on a globe, and removeSource discards parsed geometry so
// re-enabling a layer would refetch it. Hidden sources hold an empty
// collection, so the idle cost is a handful of empty objects.
export function situationLayers(map, palette, catalogue) {
  const created = []
  const painted = []

  for (const layer of catalogue || []) {
    if (!layer || !layer.entitled || !layer.available) continue

    const spec = specFor(layer)
    if (!spec) continue

    const sourceId = situationSourceId(layer.id)
    if (map.getSource(sourceId)) continue
    map.addSource(sourceId, { type: 'geojson', data: EMPTY })

    const key = spec.palette || paletteKeyFor(layer)
    const colour = severityColor(palette)
    const [first, second] = layerIdsFor(layer.id, spec)

    if (spec.type === 'fill') {
      map.addLayer({
        id: first,
        type: 'fill',
        source: sourceId,
        layout: { visibility: 'none' },
        paint: { 'fill-color': palette[key], 'fill-opacity': 0.18 },
      })
      map.addLayer({
        id: second,
        type: 'line',
        source: sourceId,
        layout: { visibility: 'none' },
        paint: { 'line-color': palette[key], 'line-width': 1.2, 'line-opacity': 0.9 },
      })
      painted.push({ id: first, property: 'fill-color', key })
      painted.push({ id: second, property: 'line-color', key })
    } else {
      map.addLayer({
        id: first,
        type: 'circle',
        source: sourceId,
        layout: { visibility: 'none' },
        paint: {
          'circle-color': colour,
          'circle-radius': radiusFor(spec),
          'circle-opacity': 0.85,
          'circle-stroke-width': 0.6,
          'circle-stroke-color': palette['label-halo'] || '#0b1220',
        },
      })
      painted.push({ id: first, property: 'circle-color', key, graded: true })
    }

    created.push(layer.id)
  }

  createdLayers.set(map, painted)
  return created
}

// setLayerVisibility shows or hides every MapLibre layer belonging to one
// catalogue layer. Missing layers are ignored so a toggle for something the
// caller cannot see is harmless rather than an exception.
export function setLayerVisibility(map, layerID, visible) {
  const base = situationSourceId(layerID)
  const visibility = visible ? 'visible' : 'none'

  for (const suffix of ['-fill', '-line', '-dot']) {
    const id = base + suffix
    if (map.getLayer(id)) map.setLayoutProperty(id, 'visibility', visibility)
  }
}

// setLayerData replaces one layer's features.
export function setLayerData(map, layerID, collection) {
  const source = map.getSource(situationSourceId(layerID))
  if (source && typeof source.setData === 'function') {
    source.setData(collection || EMPTY)
  }
}

// repaintSituation recolours every layer this module created.
//
// Point layers keep a data-driven match expression rather than a flat colour:
// setting a flat value would silently collapse the severity grading, so those
// are rebuilt from the new palette instead of assigned from it.
export function repaintSituation(map, palette) {
  if (!map || !map.getLayer) return

  const painted = createdLayers.get(map)
  if (!painted) return

  const graded = severityColor(palette)
  for (const { id, property, key, graded: isGraded } of painted) {
    if (!map.getLayer(id)) continue
    map.setPaintProperty(id, property, isGraded ? graded : palette[key])
  }
}

// refreshSituationLayer fetches a layer's features and hands them to its
// source.
//
// Called when a layer is first switched on rather than at load, so a page with
// fifteen registered layers costs one request per layer the viewer actually
// wants rather than fifteen up front. The browser's own conditional GET against
// the endpoint's ETag absorbs most of the repeat cost.
export async function refreshSituationLayer(map, layerID, options = {}) {
  const source = map.getSource(situationSourceId(layerID))
  if (!source || typeof source.setData !== 'function') return

  const params = new URLSearchParams()
  if (options.bbox) params.set('bbox', options.bbox)

  const query = params.toString()
  const url = `/situation/layers/${encodeURIComponent(layerID)}.geojson${query ? `?${query}` : ''}`

  try {
    const response = await fetch(url, { credentials: 'same-origin' })
    if (!response.ok) return
    source.setData(await response.json())
  } catch (error) {
    // A layer that will not load must not take the map down with it.
    console.warn('[skyvisor] situation layer failed to load', layerID, error)
  }
}

// shouldRefreshLayer decides whether a layer-updated event is worth acting on.
//
// The event names one layer precisely, so a busy hazard feed does not make
// every other source refetch itself, and a layer nobody has switched on is left
// alone — refetching it spends the request for a picture that is never drawn.
export function shouldRefreshLayer(payload, visibleLayers) {
  const layerId = payload?.layer
  if (!layerId) return false
  return visibleLayers.has(layerId)
}

// alertMessage renders an alert payload as a sentence.
//
// Falls back to naming the layer when the headline is missing: the server
// should always send one, but a toast reading "undefined" is a worse failure
// than a generic sentence.
export function alertMessage(payload) {
  if (!payload) return ''

  const headline = String(payload.headline || '').trim()
  if (headline) return headline

  const layer = String(payload.layer || 'situation').trim()
  const severity = String(payload.max_severity || '').trim()
  return severity ? `New ${severity} observation in ${layer}` : `New observation in ${layer}`
}

// watchSituationEvents wires the live stream to a map.
//
// Returns a release function; the caller drops it when the map goes away, or
// the subscription outlives the page and the handler fires against a torn-down
// renderer.
export function watchSituationEvents(map, visibleLayers, options = {}) {
  const { subscribe, onAlert } = options
  if (typeof subscribe !== 'function') return () => {}

  const releaseUpdates = subscribe(['situation.layer.updated'], (event) => {
    const payload = event?.situation
    if (!shouldRefreshLayer(payload, visibleLayers)) return
    void refreshSituationLayer(map, payload.layer)
  })

  const releaseAlerts = subscribe(['situation.alert'], (event) => {
    const message = alertMessage(event?.situation)
    if (message && typeof onAlert === 'function') onAlert(message, event.situation)
  })

  return () => {
    releaseUpdates()
    releaseAlerts()
  }
}
