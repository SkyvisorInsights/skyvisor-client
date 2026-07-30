// Route geometry helpers.
//
// FIXME(P1): despite the name these interpolate linearly in lon/lat, which is a
// rhumb-ish line, not a great circle. LIS->JFK renders visibly wrong and any
// route crossing the antimeridian produces a line that wraps the long way round
// the map. P1 replaces both with the `arc` package's GreatCircle/interpolate,
// which also returns MultiLineString for antimeridian-crossing routes.
// Moved here unchanged so the bundle split stays behaviour-neutral.

export function greatCirclePoints(from, to, steps = 64) {
  const points = []
  for (let i = 0; i <= steps; i += 1) {
    const t = i / steps
    points.push([from[0] + (to[0] - from[0]) * t, from[1] + (to[1] - from[1]) * t])
  }
  return points
}

export function routePointAtProgress(from, to, progress) {
  const points = greatCirclePoints(from, to, 48)
  const index = Math.min(points.length - 1, Math.max(0, Math.round((progress / 100) * (points.length - 1))))
  return points[index]
}
