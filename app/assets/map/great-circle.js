import { GreatCircle } from 'arc'

// Real spherical geometry, replacing the linear lon/lat interpolation this file
// used to contain. That produced visibly wrong routes (LIS->JFK bowed the wrong
// way) and drew a line straight across the map for anything crossing the
// antimeridian.

// Builds a great-circle arc as a GeoJSON Feature.
//
// NOTE: geometry may be LineString *or* MultiLineString — the arc library
// splits routes that cross the antimeridian into two parts. MapLibre line
// layers accept both, but any caller reaching into geometry.coordinates[0]
// must handle the split form.
export function arcFeature(from, to, properties = {}, points = 64) {
  const generator = new GreatCircle(
    { x: from[0], y: from[1] },
    { x: to[0], y: to[1] },
    properties,
  )
  return generator.Arc(points).json()
}

// Position at a fraction (0..1) along the great circle between two points.
// GreatCircle.interpolate returns a plain [lon, lat] array, not a {x, y} object.
export function pointAtProgress(from, to, fraction) {
  const clamped = Math.min(1, Math.max(0, fraction))
  const generator = new GreatCircle(
    { x: from[0], y: from[1] },
    { x: to[0], y: to[1] },
  )
  return generator.interpolate(clamped)
}

// Convenience for callers that want plain coordinates (e.g. to compute bounds).
// Uses the same spherical interpolation.
export function arcPoints(from, to, steps = 64) {
  const generator = new GreatCircle(
    { x: from[0], y: from[1] },
    { x: to[0], y: to[1] },
  )
  const points = []
  for (let i = 0; i <= steps; i += 1) {
    points.push(generator.interpolate(i / steps))
  }
  return points
}
