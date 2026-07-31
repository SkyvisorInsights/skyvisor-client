import { reducedMotion } from '../../core/env.js'

// Makes the Activities drawer the working keyboard equivalent of the globe.
//
// A WebGL canvas is not an accessible control. Rather than pretend otherwise,
// every row in the drawer carries the coordinates of the thing it describes;
// focusing or activating a row moves the globe there and announces the move in
// a polite live region. The data stays fully reachable without ever touching
// the canvas.

function liveRegion() {
  let node = document.getElementById('globe-live-region')
  if (node) return node
  node = document.createElement('p')
  node.id = 'globe-live-region'
  node.className = 'sr-only'
  node.setAttribute('aria-live', 'polite')
  document.body.appendChild(node)
  return node
}

function announce(message) {
  if (!message) return
  const region = liveRegion()
  // Reassigning identical text does not re-announce, so clear first.
  region.textContent = ''
  window.setTimeout(() => {
    region.textContent = message
  }, 30)
}

function describe(row) {
  const cells = row.querySelectorAll('td')
  const parts = []
  cells.forEach((cell) => {
    const text = cell.textContent.trim()
    if (text && text !== '—') parts.push(text)
  })
  return parts.join(', ')
}

export function initGlobeFocus(root = document) {
  const scope = root && root.querySelectorAll ? root : document
  scope.querySelectorAll('[data-globe-focus]').forEach((row) => {
    if (row.dataset.globeFocusReady === 'true') return
    row.dataset.globeFocusReady = 'true'

    const lon = Number(row.dataset.globeLon)
    const lat = Number(row.dataset.globeLat)
    const hasPoint = Number.isFinite(lon) && Number.isFinite(lat)
      && row.dataset.globeLon !== undefined && row.dataset.globeLon !== ''

    const go = () => {
      announce(describe(row))
      if (!hasPoint) return
      const canvas = document.querySelector('[data-globe-canvas]')
      const map = canvas && canvas._skyvisorMap
      if (!map) return
      map.flyTo({
        center: [lon, lat],
        zoom: Math.max(map.getZoom(), 2.6),
        duration: reducedMotion() ? 0 : 900,
      })
    }

    row.addEventListener('focus', go)
    row.addEventListener('click', go)
    row.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        go()
      }
    })
  })
}
