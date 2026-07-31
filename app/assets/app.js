import 'htmx.org'
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

document.body.addEventListener('htmx:afterSwap', (event) => {
  boot(event.detail.target)
  // A swap that carried a new globe envelope updates the existing map in place.
  if (window.SkyVisorMap && document.getElementById('globe-bootstrap')) {
    window.SkyVisorMap.refresh(event.detail.target)
  }
})

document.body.addEventListener('htmx:beforeCleanupElement', (event) => {
  releaseLiveElement(event.detail.elt)
})

initThemeWatcher()
