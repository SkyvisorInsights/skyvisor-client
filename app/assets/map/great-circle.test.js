import { expect, test } from 'bun:test'
import { arcFeature, arcPoints, pointAtProgress } from './great-circle.js'

const LIS = [-9.1359, 38.7813]
const JFK = [-73.7781, 40.6413]
const HND = [139.7798, 35.5494]
const SFO = [-122.375, 37.6188]

test('pointAtProgress returns a real great-circle midpoint, not a linear one', () => {
  const [lon, lat] = pointAtProgress(LIS, JFK, 0.5)

  expect(Number.isFinite(lon)).toBe(true)
  expect(Number.isFinite(lat)).toBe(true)

  // The linear midpoint of these two points is ~39.71N. A great circle between
  // Lisbon and New York bows north of that by roughly five degrees. If this
  // assertion fails we have regressed to interpolating in lon/lat space.
  const linearMidLat = (LIS[1] + JFK[1]) / 2
  expect(lat).toBeGreaterThan(linearMidLat + 3)
  expect(lat).toBeCloseTo(44.5026, 3)
  expect(lon).toBeCloseTo(-40.9683, 3)
})

test('pointAtProgress hits the endpoints and clamps out-of-range input', () => {
  expect(pointAtProgress(LIS, JFK, 0)[0]).toBeCloseTo(LIS[0], 6)
  expect(pointAtProgress(LIS, JFK, 0)[1]).toBeCloseTo(LIS[1], 6)
  expect(pointAtProgress(LIS, JFK, 1)[0]).toBeCloseTo(JFK[0], 6)
  expect(pointAtProgress(LIS, JFK, 1)[1]).toBeCloseTo(JFK[1], 6)

  expect(pointAtProgress(LIS, JFK, -3)).toEqual(pointAtProgress(LIS, JFK, 0))
  expect(pointAtProgress(LIS, JFK, 42)).toEqual(pointAtProgress(LIS, JFK, 1))
})

test('arcFeature splits antimeridian-crossing routes into MultiLineString', () => {
  const pacific = arcFeature(HND, SFO)
  expect(pacific.geometry.type).toBe('MultiLineString')
  expect(pacific.geometry.coordinates.length).toBe(2)

  // A route that does not cross the antimeridian stays a simple LineString.
  const atlantic = arcFeature(LIS, JFK)
  expect(atlantic.geometry.type).toBe('LineString')
})

test('arcFeature carries properties through to the feature', () => {
  const feature = arcFeature(LIS, JFK, { id: 'LIS-JFK', risk: 'watch' })
  expect(feature.properties.id).toBe('LIS-JFK')
  expect(feature.properties.risk).toBe('watch')
})

test('arcPoints samples the sphere and returns finite coordinates', () => {
  const points = arcPoints(LIS, JFK, 16)
  expect(points.length).toBe(17)

  for (const [lon, lat] of points) {
    expect(Number.isFinite(lon)).toBe(true)
    expect(Number.isFinite(lat)).toBe(true)
    expect(Math.abs(lat)).toBeLessThanOrEqual(90)
  }

  // Same curve as pointAtProgress.
  const mid = pointAtProgress(LIS, JFK, 0.5)
  expect(points[8][0]).toBeCloseTo(mid[0], 6)
  expect(points[8][1]).toBeCloseTo(mid[1], 6)
})
