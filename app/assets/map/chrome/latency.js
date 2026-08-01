// Latency and freshness chips.
//
// Three numbers were candidates here and only two survive scrutiny:
//
//   Rejected — "SSE delivery latency", Date.now() minus the server timestamp on
//   the event. That is one clock minus a different clock. Browser clocks are
//   routinely seconds out, so the figure would be confidently wrong and
//   occasionally negative.
//
//   LINK — round trip measured with the Resource Timing API on our own request.
//   One clock, two readings, no skew, and a genuine property of the user's
//   connection.
//
//   DATA — how old the newest observed aircraft position is. This is what an
//   operator actually cares about, and it is seconds to minutes, not
//   milliseconds.
//
// Each chip says what it measures. A decorative number that measures nothing
// erodes trust in every other number on the page.

function linkLatencyFor(url) {
  const entries = performance.getEntriesByName(url)
  const entry = entries[entries.length - 1]
  if (!entry || !entry.responseStart || !entry.requestStart) return null
  const ms = Math.round(entry.responseStart - entry.requestStart)
  return ms >= 0 ? ms : null
}

function formatAge(seconds) {
  if (seconds < 60) return `${Math.max(0, Math.round(seconds))}s`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  return `${Math.round(minutes / 60)}h`
}

// Freshness thresholds mirror the dashboard's freshness badge so the two
// surfaces cannot disagree about what "stale" means.
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

export function initLatencyChips(root = document) {
  const link = root.querySelector?.('#globe-link-latency') || document.getElementById('globe-link-latency')
  const data = root.querySelector?.('#globe-data-age') || document.getElementById('globe-data-age')
  if (!link && !data) return null

  const update = (envelope, response) => {
    if (link) {
      const url = response?.url || new URL('/globe/data', location.origin).href
      const ms = linkLatencyFor(url)
      if (ms !== null) {
        link.textContent = `LINK · ${ms} ms`
        link.hidden = false
        const upstream = response?.headers?.get?.('X-Skyvisor-Upstream-Ms')
        link.title = upstream
          ? `Round trip to this server measured on your connection. The server spent ${upstream} ms calling the API.`
          : 'Round trip to this server, measured on your connection.'
      }
    }

    if (data) {
      const newest = newestObservation(envelope?.flights?.features)
      if (newest === null) {
        // No observation carries a timestamp, so there is nothing honest to
        // report. Say nothing rather than invent a freshness.
        data.textContent = 'DATA · unknown'
        data.dataset.tone = 'unknown'
        data.title = 'No aircraft position carries an observation time.'
      } else {
        const seconds = (Date.now() - newest) / 1000
        data.textContent = `DATA · ${formatAge(seconds)}`
        data.dataset.tone = ageTone(seconds)
        data.title = 'Age of the most recent aircraft position on the globe.'
      }
      data.hidden = false
    }
  }

  return update
}
