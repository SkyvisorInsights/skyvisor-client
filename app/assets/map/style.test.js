import { expect, test } from 'bun:test'
import { PALETTE_FALLBACK, PALETTE_KEYS } from './palette.js'
import { PAINT_BINDINGS, buildStyle, repaint, skySpec } from './style.js'

// The canvas colour probe needs a DOM, which bun test does not provide, so the
// resolver itself is exercised in the browser. What is testable here is that
// the fallback table is complete and that the style wiring is consistent — the
// two things that silently produce a black-on-black map when they drift.

test('every palette key has a fallback colour', () => {
  for (const key of PALETTE_KEYS) {
    expect(PALETTE_FALLBACK[key]).toBeDefined()
    expect(PALETTE_FALLBACK[key]).toMatch(/^#[0-9a-f]{6}$/i)
  }
  expect(Object.keys(PALETTE_FALLBACK).sort()).toEqual([...PALETTE_KEYS].sort())
})

test('buildStyle produces a valid style referencing self-hosted geometry', () => {
  const style = buildStyle(PALETTE_FALLBACK)

  expect(style.version).toBe(8)

  // No third-party host may appear in the style: the whole point of P2 is that
  // the basemap is self-hosted.
  const serialized = JSON.stringify(style)
  expect(serialized).not.toContain('demotiles.maplibre.org')
  expect(serialized).not.toContain('http://')
  expect(serialized).not.toContain('https://naciscdn')
  for (const source of Object.values(style.sources)) {
    expect(source.data.startsWith('/static/geo/')).toBe(true)
  }

  // Ocean must sit under land, land under boundaries.
  const ids = style.layers.map((l) => l.id)
  expect(ids.indexOf('basemap-ocean')).toBeLessThan(ids.indexOf('basemap-land'))
  expect(ids.indexOf('basemap-land')).toBeLessThan(ids.indexOf('basemap-boundary'))
})

test('every basemap layer is themeable', () => {
  const style = buildStyle(PALETTE_FALLBACK)
  const bound = new Set(PAINT_BINDINGS.map(([id]) => id))

  for (const layer of style.layers) {
    expect(bound.has(layer.id)).toBe(true)
  }
  // Bindings must reference real palette keys.
  for (const [, , key] of PAINT_BINDINGS) {
    expect(PALETTE_KEYS).toContain(key)
  }
})

test('repaint sets every bound layer that exists and skips the rest', () => {
  const calls = []
  const present = new Set(['basemap-ocean', 'basemap-land'])
  const fakeMap = {
    getLayer: (id) => (present.has(id) ? { id } : undefined),
    setPaintProperty: (id, prop, value) => calls.push([id, prop, value]),
    setSky: () => {},
  }

  repaint(fakeMap, PALETTE_FALLBACK)

  expect(calls).toEqual([
    ['basemap-ocean', 'background-color', PALETTE_FALLBACK.ocean],
    ['basemap-land', 'fill-color', PALETTE_FALLBACK.land],
  ])
})

test('repaint tolerates a map without sky support', () => {
  const fakeMap = { getLayer: () => undefined, setPaintProperty: () => {} }
  expect(() => repaint(fakeMap, PALETTE_FALLBACK)).not.toThrow()
  expect(() => repaint(null, PALETTE_FALLBACK)).not.toThrow()
})

test('skySpec uses palette colours', () => {
  const sky = skySpec(PALETTE_FALLBACK)
  expect(sky['sky-color']).toBe(PALETTE_FALLBACK.ocean)
  expect(sky['horizon-color']).toBe(PALETTE_FALLBACK.atmosphere)
})
