import 'htmx.org'
// htmx 4 can morph rather than replace, and a morph discards Alpine component
// state. This is part of the stack, not a migration bridge.
import 'htmx.org/dist/ext/hx-alpine-compat.js'
import Alpine from 'alpinejs'
import { applyTheme, currentTheme, initThemeWatcher } from './core/theme.js'
import { initMotion } from './core/motion.js'
import { initLiveRefresh, releaseLiveElement } from './core/live.js'

Alpine.data('appShell', () => ({
  menuOpen: false,
  theme: currentTheme(),
  cycleTheme() {
    const modes = ['system', 'light', 'dark']
    this.theme = modes[(modes.indexOf(this.theme) + 1) % modes.length]
    applyTheme(this.theme)
  },
}))

Alpine.data('flightSearch', () => ({
  flight: '',
  submit() {
    this.flight = this.flight.trim().toUpperCase().replace(/\s+/g, '')
  },
}))

const globeProjectionKey = 'skyvisor-globe-projection'

  // The situation layer rail. Toggling drives MapLibre visibility directly
  // rather than re-rendering, and the chosen set is remembered per viewer so a
  // reload does not reset the map to blank.
  Alpine.data('situationPage', () => ({
    enabled: [],
    alerts: [],

    init() {
      try {
        this.enabled = JSON.parse(localStorage.getItem('skyvisor:situation:layers') || '[]')
      } catch {
        this.enabled = []
      }
      this.$nextTick(() => this.enabled.forEach((id) => this.apply(id, true)))

      // Alerts are raised by the map module and rendered here. Capped and
      // newest-first: an unbounded list turns a busy hour into a wall that
      // hides the map it is describing.
      window.addEventListener('skyvisor:situation-alert', (event) => {
        const message = event.detail?.message
        if (!message) return
        this.alerts = [{ id: `${Date.now()}-${this.alerts.length}`, message }, ...this.alerts].slice(0, 4)
      })
    },

    dismiss(id) {
      this.alerts = this.alerts.filter((alert) => alert.id !== id)
    },

    toggle(layerId, on) {
      this.enabled = on
        ? [...new Set([...this.enabled, layerId])]
        : this.enabled.filter((id) => id !== layerId)
      try {
        localStorage.setItem('skyvisor:situation:layers', JSON.stringify(this.enabled))
      } catch {
        // A viewer with storage disabled still gets a working map; only the
        // remembered selection is lost.
      }
      this.apply(layerId, on)
    },

    apply(layerId, on) {
      window.dispatchEvent(new CustomEvent('skyvisor:situation-layer', {
        detail: { layerId, visible: on },
      }))
    },
  }))

  Alpine.data('globeProjection', () => ({
  // Seeded from the server-rendered pressed state so the button and the map
  // agree before any JavaScript runs.
  projection: document.querySelector('[data-globe-canvas]')?.dataset.globeProjection === '2d' ? 'mercator' : 'globe',
  set(value) {
    if (this.projection === value) return
    this.projection = value
    localStorage.setItem(globeProjectionKey, value === 'mercator' ? '2d' : 'globe')
    window.dispatchEvent(new CustomEvent('sky:projection', { detail: value === 'mercator' ? '2d' : 'globe' }))
  },
}))

const globeDrawerKey = 'skyvisor-globe-drawer'

Alpine.data('globeDrawer', () => ({
  open: localStorage.getItem(globeDrawerKey) !== 'closed',
  init() {
    this.$watch('open', (value) => {
      localStorage.setItem(globeDrawerKey, value ? 'open' : 'closed')
    })
  },
}))

// --- lazy map bundle ------------------------------------------------------
//
// MapLibre and its layer code live in a separate bundle that is only fetched
// when a page actually renders a map container. Plain script injection is used
// rather than esbuild's ESM code splitting because the codebase (and several
// inline templ script blocks) depend on htmx and Alpine being globals, which
// <script type="module"> would break.

const MAP_BUNDLE_URL = '/static/js/map.js'
let mapBundle = null

function ensureMapBundle() {
  if (mapBundle) return mapBundle
  mapBundle = new Promise((resolve, reject) => {
    if (window.SkyVisorMap) {
      resolve(window.SkyVisorMap)
      return
    }
    const script = document.createElement('script')
    script.src = MAP_BUNDLE_URL
    script.async = true
    script.onload = () => {
      if (window.SkyVisorMap) resolve(window.SkyVisorMap)
      else reject(new Error('map bundle loaded without SkyVisorMap'))
    }
    script.onerror = () => reject(new Error(`failed to load ${MAP_BUNDLE_URL}`))
    document.head.appendChild(script)
  })
  return mapBundle
}

function hasMapContainer(root) {
  if (root.nodeType === 1 && typeof root.matches === 'function' && root.matches('[data-skyvisor-map]')) return true
  return typeof root.querySelector === 'function' && root.querySelector('[data-skyvisor-map]') !== null
}

function initMaps(root = document) {
  if (!hasMapContainer(root)) return
  ensureMapBundle()
    .then((bundle) => bundle.init(root))
    .catch((error) => {
      // The server-rendered fallback stays visible; nothing else to do.
      console.error('[skyvisor] map bundle unavailable', error)
    })
}

// --- boot -----------------------------------------------------------------

window.Alpine = Alpine
window.SkyVisor = { applyTheme, initMaps, initLiveRefresh, initMotion }

Alpine.start()

function boot(root = document) {
  initMaps(root)
  initLiveRefresh(root)
  initMotion(root)
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => boot())
} else {
  boot()
}

// htmx 4 renamed these events and moved the swap target: detail.target is gone,
// and the target now lives on the request context as detail.ctx.target. The
// event's own target is the element that made the request, not the one swapped.
document.body.addEventListener('htmx:after:swap', (event) => {
  boot(event.detail.ctx.target)
  // A swap that carried a new globe envelope updates the existing map in place.
  if (window.SkyVisorMap && document.getElementById('globe-bootstrap')) {
    window.SkyVisorMap.refresh(event.detail.ctx.target)
  }
})

// htmx:before:cleanup carries no detail at all; the element being torn down is
// the event target.
document.body.addEventListener('htmx:before:cleanup', (event) => {
  releaseLiveElement(event.target)
})

initThemeWatcher()
