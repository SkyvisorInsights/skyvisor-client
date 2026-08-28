// Render specs for situation layers.
//
// Keyed by layer id where a layer deserves bespoke treatment, with a fallback
// derived from the catalogue entry's geometry and category. The fallback is
// what lets a layer added server-side appear without a client release, which is
// the whole reason the catalogue is fetched rather than hardcoded.

// Palette key per category, so a new category needs one token rather than a
// per-layer colour decision.
const CATEGORY_PALETTE = {
  hazard: 'situation-hazard',
  seismic: 'situation-seismic',
  fire: 'situation-fire',
  conflict: 'situation-conflict',
  news: 'situation-news',
  market: 'situation-market',
  air: 'situation-air',
}

export const DEFAULT_PALETTE_KEY = 'situation-hazard'

// Bespoke specs. A radiusMetric scales the circle by a number carried in the
// feature's properties, so a magnitude 7 quake does not look like a magnitude 2.
export const LAYER_SPECS = {
  'quake.usgs': { type: 'circle', radiusMetric: 'magnitude', palette: 'situation-seismic' },
  'fire.firms': { type: 'circle', radiusMetric: 'frp_mw', palette: 'situation-fire' },
  // Natural events span wildfires, storms and ice. The layer is one source, so
  // it gets one colour; the per-feature severity expression carries the rest.
  eonet: { type: 'circle', palette: 'situation-hazard' },
}

export function paletteKeyFor(layer) {
  return CATEGORY_PALETTE[layer && layer.category] || DEFAULT_PALETTE_KEY
}

// specFor resolves a catalogue entry to a render spec.
//
// Returns null for a layer with no geometry: news and indicator layers are real
// layers, they simply belong in a rail rather than on the map.
export function specFor(layer) {
  if (!layer || !layer.id) return null

  const bespoke = LAYER_SPECS[layer.id]
  if (bespoke) return bespoke

  const palette = paletteKeyFor(layer)
  if (layer.geometry === 'polygon') return { type: 'fill', palette }
  if (layer.geometry === 'point') return { type: 'circle', palette }
  return null
}
