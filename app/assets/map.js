// Map bundle entry point.
//
// Loaded lazily by app.js, only on pages that actually render a map container.
// MapLibre is ~1.1 MB, so keeping it out of the main bundle removes that weight
// from every other page on the site.

import maplibregl from 'maplibre-gl'
import { initMaps, refreshGlobe, teardownMap, hasMap } from './map/registry.js'
import { initFleetFilmstrip } from './map/chrome/filmstrip.js'
import { initGlobeFocus } from './map/chrome/focus.js'
import { initGlobeLive } from './map/chrome/live.js'
import { initLatencyChips } from './map/chrome/latency.js'
import { initGlobeReadout, initProjectionToggle } from './map/chrome/readout.js'

let releaseLive = null

function init(root = document) {
  initMaps(root)
  initFleetFilmstrip(root)
  initGlobeFocus(root)
  startLive()
}

// Live flight positions are wired once the globe map exists. The map is created
// asynchronously inside initMaps, so wait for it rather than assuming.
function startLive() {
  const canvas = document.querySelector('[data-globe-canvas]')
  if (!canvas || releaseLive) return
  const attach = () => {
    if (releaseLive) return
    const onEnvelope = initLatencyChips(document)
    releaseLive = initGlobeLive(canvas, { onEnvelope })
    initGlobeReadout(document)
    initProjectionToggle()
  }
  if (canvas._skyvisorMap) {
    canvas._skyvisorMap.once('load', attach)
  } else {
    // initMaps has not created it yet; retry on the next frame.
    requestAnimationFrame(startLive)
  }
}

// Called after an htmx swap replaced the globe panels. The canvas itself is
// outside the swapped region, so the existing map is updated in place rather
// than rebuilt — recreating a WebGL context on every data tick would be both
// slow and visibly jarring.
function refresh(root = document) {
  const canvas = document.querySelector('[data-globe-canvas]')
  if (!canvas || !canvas._skyvisorMap) return
  refreshGlobe(canvas)
  initGlobeFocus(root)
}

window.maplibregl = maplibregl
window.SkyVisorMap = { init, refresh, initMaps, initFleetFilmstrip, teardownMap, hasMap }
