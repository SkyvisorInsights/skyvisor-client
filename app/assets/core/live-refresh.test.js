import { describe, expect, test } from 'bun:test'
import { refreshEventsFor } from './live.js'

describe('live refresh event selection', () => {
  test('defaults to the flight events', () => {
    // Every existing consumer — the dashboard, the globe panels, the track
    // page — relies on this default and declares no events of its own.
    const events = refreshEventsFor({ dataset: {} })
    expect(events).toContain('flight.updated')
    expect(events).toContain('gate.changed')
  })

  test('an element may name its own events', () => {
    // The news rail cares about situation events and nothing else. Subscribing
    // it to the flight events would refetch sixty stories every time an
    // unrelated aircraft changed gate.
    const events = refreshEventsFor({ dataset: { liveRefreshEvents: 'situation.news' } })
    expect(events).toEqual(['situation.news'])
  })

  test('a list is split and trimmed', () => {
    const events = refreshEventsFor({ dataset: { liveRefreshEvents: ' situation.news , situation.alert ' } })
    expect(events).toEqual(['situation.news', 'situation.alert'])
  })

  test('an empty declaration falls back to the default', () => {
    // A blank attribute is a template mistake, not a request to subscribe to
    // nothing, and silently subscribing to nothing is invisible.
    const events = refreshEventsFor({ dataset: { liveRefreshEvents: '  ' } })
    expect(events).toContain('flight.updated')
  })
})
