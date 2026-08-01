import { expect, test } from 'bun:test'

// The freshness maths is the part that can quietly become a lie, so it is
// extracted here and asserted directly. The DOM wiring needs a browser and is
// checked by hand.

function formatAge(seconds) {
  if (seconds < 60) return `${Math.max(0, Math.round(seconds))}s`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  return `${Math.round(minutes / 60)}h`
}

function ageTone(seconds) {
  if (seconds <= 120) return 'fresh'
  if (seconds <= 600) return 'ageing'
  return 'stale'
}

function newestObservation(features) {
  let newest = null
  for (const feature of features || []) {
    const raw = feature.properties?.updated_at
    if (!raw) continue
    const parsed = Date.parse(raw)
    if (Number.isNaN(parsed)) continue
    if (newest === null || parsed > newest) newest = parsed
  }
  return newest
}

test('age formatting stays readable across scales', () => {
  expect(formatAge(0)).toBe('0s')
  expect(formatAge(14.4)).toBe('14s')
  expect(formatAge(59)).toBe('59s')
  expect(formatAge(60)).toBe('1m')
  expect(formatAge(3540)).toBe('59m')
  // 3599s rounds to 60 minutes, which reads better promoted to hours.
  expect(formatAge(3599)).toBe('1h')
  expect(formatAge(7200)).toBe('2h')
})

test('a negative age never renders as a negative number', () => {
  // Clock skew between server and browser can make an observation look like it
  // is from the future. Reporting "-3s old" would be visibly broken.
  expect(formatAge(-5)).toBe('0s')
})

test('freshness tone matches the dashboard thresholds', () => {
  expect(ageTone(0)).toBe('fresh')
  expect(ageTone(120)).toBe('fresh')
  expect(ageTone(121)).toBe('ageing')
  expect(ageTone(600)).toBe('ageing')
  expect(ageTone(601)).toBe('stale')
})

test('newest observation ignores missing and malformed timestamps', () => {
  const features = [
    { properties: { updated_at: '2026-07-31T10:00:00Z' } },
    { properties: {} },
    { properties: { updated_at: 'not a date' } },
    { properties: { updated_at: '2026-07-31T10:05:00Z' } },
  ]
  expect(newestObservation(features)).toBe(Date.parse('2026-07-31T10:05:00Z'))
})

// No timestamps at all must report unknown rather than defaulting to "now",
// which would claim the data is fresh when we have no idea.
test('no usable timestamps reports null, not now', () => {
  expect(newestObservation([{ properties: {} }, { properties: { updated_at: '' } }])).toBeNull()
  expect(newestObservation([])).toBeNull()
  expect(newestObservation(undefined)).toBeNull()
})
