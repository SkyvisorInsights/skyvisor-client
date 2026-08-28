import { describe, expect, test } from 'bun:test'
import { PALETTE_FALLBACK } from '../palette.js'
import { specFor } from './situation-layers.js'
import {
  alertMessage,
  repaintSituation,
  severityColor,
  shouldRefreshLayer,
  situationLayers,
  situationSourceId,
  setLayerVisibility,
} from './situation.js'

// A stand-in for the parts of the MapLibre API this module touches. Asserting
// against a fake map is the only way to test layer wiring without WebGL.
function fakeMap() {
  const sources = new Map()
  const layers = new Map()
  return {
    sources,
    layers,
    addSource: (id, spec) => sources.set(id, spec),
    getSource: (id) => sources.get(id),
    addLayer: (spec) => layers.set(spec.id, spec),
    getLayer: (id) => layers.get(id),
    setLayoutProperty: (id, property, value) => {
      const layer = layers.get(id)
      if (!layer) throw new Error(`no such layer ${id}`)
      layer.layout = { ...(layer.layout || {}), [property]: value }
    },
    setPaintProperty: (id, property, value) => {
      const layer = layers.get(id)
      if (!layer) throw new Error(`no such layer ${id}`)
      layer.paint = { ...(layer.paint || {}), [property]: value }
    },
    setProjection: () => {},
  }
}

const catalogue = [
  { id: 'sigmet.intl', category: 'hazard', geometry: 'polygon', entitled: true, available: true },
  { id: 'quake.usgs', category: 'seismic', geometry: 'point', entitled: true, available: true },
]

describe('situation map mode', () => {
  test('adds one source per entitled layer', () => {
    const map = fakeMap()
    situationLayers(map, PALETTE_FALLBACK, catalogue)

    expect(map.getSource(situationSourceId('sigmet.intl'))).toBeTruthy()
    expect(map.getSource(situationSourceId('quake.usgs'))).toBeTruthy()
  })

  test('seeds every source with an empty FeatureCollection', () => {
    const map = fakeMap()
    situationLayers(map, PALETTE_FALLBACK, catalogue)

    // MapLibre treats a null or absent features array as a parse error and
    // drops the source silently, so the initial state has to be a valid empty
    // collection rather than nothing at all.
    const source = map.getSource(situationSourceId('sigmet.intl'))
    expect(source.data.type).toBe('FeatureCollection')
    expect(source.data.features).toEqual([])
  })

  test('adds every layer hidden', () => {
    const map = fakeMap()
    situationLayers(map, PALETTE_FALLBACK, catalogue)

    // Layers are created once and toggled by visibility. Adding and removing
    // them on toggle re-sorts and re-validates the whole style, which visibly
    // hitches on a globe, and removing a source discards its parsed geometry.
    for (const layer of map.layers.values()) {
      expect(layer.layout.visibility).toBe('none')
    }
  })

  test('does not create layers the caller is not entitled to', () => {
    const map = fakeMap()
    situationLayers(map, PALETTE_FALLBACK, [
      { id: 'fire.firms', category: 'fire', geometry: 'point', entitled: false, available: true },
    ])

    expect(map.sources.size).toBe(0)
    expect(map.layers.size).toBe(0)
  })

  test('renders a polygon layer as both fill and outline', () => {
    const map = fakeMap()
    situationLayers(map, PALETTE_FALLBACK, catalogue)

    const ids = [...map.layers.keys()].filter((id) => id.includes('sigmet-intl'))
    const types = ids.map((id) => map.getLayer(id).type).sort()
    // A translucent fill alone reads as a smudge at low zoom; the outline is
    // what makes the hazard boundary legible.
    expect(types).toEqual(['fill', 'line'])
  })

  test('toggles visibility rather than recreating layers', () => {
    const map = fakeMap()
    situationLayers(map, PALETTE_FALLBACK, catalogue)
    const before = map.layers.size

    setLayerVisibility(map, 'sigmet.intl', true)
    for (const [id, layer] of map.layers) {
      if (id.includes('sigmet-intl')) expect(layer.layout.visibility).toBe('visible')
    }

    setLayerVisibility(map, 'sigmet.intl', false)
    for (const [id, layer] of map.layers) {
      if (id.includes('sigmet-intl')) expect(layer.layout.visibility).toBe('none')
    }

    expect(map.layers.size).toBe(before)
  })

  test('colours by severity with a data-driven expression', () => {
    const expression = severityColor(PALETTE_FALLBACK)

    // A flat colour here would make every hazard look alike, and a later
    // setPaintProperty with a flat value would destroy the expression.
    expect(Array.isArray(expression)).toBe(true)
    expect(expression[0]).toBe('match')
    expect(expression).toContain('severe')
  })

  test('repaints every layer it created when the theme changes', () => {
    // The generic repaint() in style.js walks a static binding list, which
    // cannot know about a layer the server added to the catalogue after this
    // client shipped. Situation layers therefore own their own repaint, and
    // every layer it created must be covered — a missed one keeps its old
    // colour through a theme switch and nothing reports it.
    const map = fakeMap()
    situationLayers(map, PALETTE_FALLBACK, catalogue)

    const light = { ...PALETTE_FALLBACK, 'situation-hazard': '#111111', 'situation-seismic': '#222222' }
    repaintSituation(map, light)

    const fill = map.getLayer('situation-sigmet-intl-fill')
    expect(fill.paint['fill-color']).toBe('#111111')
    const line = map.getLayer('situation-sigmet-intl-line')
    expect(line.paint['line-color']).toBe('#111111')

    // Point layers are graded by severity, so their colour stays an expression
    // rather than collapsing to the flat palette value.
    const dot = map.getLayer('situation-quake-usgs-dot')
    expect(Array.isArray(dot.paint['circle-color'])).toBe(true)
  })

  test('falls back to a spec derived from geometry for an unknown layer', () => {
    // The catalogue is served by the API, so a layer added server-side must
    // render without waiting for a client release.
    const spec = specFor({ id: 'brand.new', category: 'hazard', geometry: 'point' })
    expect(spec).toBeTruthy()
    expect(spec.type).toBe('circle')
  })
})

describe('shipped layers', () => {
  // The three layers the API serves today. If one stops resolving to a spec it
  // silently vanishes from the map rather than erroring, so this is the check
  // that would catch a renamed layer id.
  const shipped = [
    { id: 'sigmet.intl', category: 'hazard', geometry: 'polygon' },
    { id: 'quake.usgs', category: 'seismic', geometry: 'point' },
    { id: 'eonet', category: 'hazard', geometry: 'point' },
  ]

  test('each resolves to a render spec', () => {
    for (const layer of shipped) {
      const spec = specFor(layer)
      expect(spec).toBeTruthy()
      expect(['fill', 'circle']).toContain(spec.type)
    }
  })

  test('polygon and point layers render differently', () => {
    expect(specFor(shipped[0]).type).toBe('fill')
    expect(specFor(shipped[1]).type).toBe('circle')
  })

  test('earthquakes are sized by magnitude', () => {
    // A magnitude 6 and a magnitude 2.5 drawn at the same radius makes the
    // layer decorative rather than informative.
    expect(specFor(shipped[1]).radiusMetric).toBe('magnitude')
  })

  test('all three create layers on a map', () => {
    const map = fakeMap()
    const created = situationLayers(
      map,
      PALETTE_FALLBACK,
      shipped.map((layer) => ({ ...layer, entitled: true, available: true })),
    )
    expect(created).toHaveLength(3)
    // The polygon layer contributes a fill and a line; the two point layers one
    // circle each.
    expect(map.layers.size).toBe(4)
  })
})

describe('live situation events', () => {
  test('a layer update only refreshes the layer that moved', () => {
    // The payload names one layer precisely so a busy hazard feed does not make
    // every other source refetch itself.
    const visible = new Set(['sigmet.intl', 'quake.usgs.m45'])
    expect(shouldRefreshLayer({ layer: 'sigmet.intl' }, visible)).toBe(true)
    expect(shouldRefreshLayer({ layer: 'quake.usgs.m45' }, visible)).toBe(true)
    expect(shouldRefreshLayer({ layer: 'eonet' }, visible)).toBe(false)
  })

  test('a hidden layer is not refreshed', () => {
    // Refetching a layer nobody is looking at spends the request and the
    // bandwidth for a picture that is never drawn.
    expect(shouldRefreshLayer({ layer: 'gdacs' }, new Set())).toBe(false)
  })

  test('a malformed payload refreshes nothing', () => {
    expect(shouldRefreshLayer(null, new Set(['gdacs']))).toBe(false)
    expect(shouldRefreshLayer({}, new Set(['gdacs']))).toBe(false)
  })

  test('an alert is described for a human, not a machine', () => {
    const message = alertMessage({
      layer: 'gdacs',
      headline: 'Red cyclone alert in Fiji',
      max_severity: 'severe',
      score: 95,
    })
    expect(message).toContain('Red cyclone alert in Fiji')
  })

  test('an alert with no headline still says something useful', () => {
    // The server should always send one, but a toast reading "undefined" is a
    // worse failure than a generic sentence.
    const message = alertMessage({ layer: 'gdacs', max_severity: 'severe', score: 95 })
    expect(message).toBeTruthy()
    expect(message).not.toContain('undefined')
    expect(message.toLowerCase()).toContain('gdacs')
  })

  test('an alert payload that is missing entirely is ignored', () => {
    expect(alertMessage(null)).toBe('')
    expect(alertMessage(undefined)).toBe('')
  })
})
