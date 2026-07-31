import { expect, test } from 'bun:test'
import { collectionsFrom } from './globe.js'

// collectionsFrom is the seam between the server envelope and the map layers.
// Every layer reads from one of the three collections it returns, so a missing
// or malformed envelope must still yield something setData can accept.

test('an envelope passes its three collections through', () => {
  const envelope = {
    routes: { type: 'FeatureCollection', features: [{ id: 'r' }] },
    hubs: { type: 'FeatureCollection', features: [{ id: 'h' }] },
    flights: { type: 'FeatureCollection', features: [{ id: 'f' }] },
  }

  const data = collectionsFrom(envelope)
  expect(data.routes.features).toHaveLength(1)
  expect(data.hubs.features).toHaveLength(1)
  expect(data.flights.features).toHaveLength(1)
})

test('a partial envelope fills the gaps with empty collections, never undefined', () => {
  const data = collectionsFrom({ routes: { type: 'FeatureCollection', features: [] } })

  for (const key of ['routes', 'hubs', 'flights']) {
    expect(data[key]).toBeDefined()
    expect(data[key].type).toBe('FeatureCollection')
    expect(Array.isArray(data[key].features)).toBe(true)
  }
})

// No envelope means the marketing globe, which still has to draw something.
test('no envelope falls back to the marketing globe', () => {
  const data = collectionsFrom(null)

  expect(data.routes.features.length).toBeGreaterThan(0)
  expect(data.hubs.features.length).toBeGreaterThan(0)
  expect(data.flights.features).toHaveLength(0)
})

test('marketing routes are real great-circle geometry with layer properties', () => {
  const data = collectionsFrom(null)

  for (const feature of data.routes.features) {
    expect(['LineString', 'MultiLineString']).toContain(feature.geometry.type)
    // The arc layers read these, so the fallback must carry them too or the
    // marketing globe renders with the wrong width and colour.
    expect(feature.properties.risk).toBe('ok')
    expect(typeof feature.properties.flights).toBe('number')
  }

  // Hub sizing reads `traffic`.
  for (const feature of data.hubs.features) {
    expect(typeof feature.properties.traffic).toBe('number')
  }
})

// The Tokyo-San Francisco leg crosses the antimeridian, so at least one
// marketing route must be split rather than drawn across the whole map.
test('marketing routes split at the antimeridian', () => {
  const data = collectionsFrom(null)
  const split = data.routes.features.filter((f) => f.geometry.type === 'MultiLineString')
  expect(split.length).toBeGreaterThan(0)
})
