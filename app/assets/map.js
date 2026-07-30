// Map bundle entry point.
//
// Loaded lazily by app.js, only on pages that actually render a map container.
// MapLibre is ~1.1 MB, so keeping it out of the main bundle removes that weight
// from every other page on the site.

import maplibregl from 'maplibre-gl'
import { initMaps, teardownMap, hasMap } from './map/registry.js'
import { initFleetFilmstrip } from './map/chrome/filmstrip.js'

function init(root = document) {
  initMaps(root)
  initFleetFilmstrip(root)
}

window.maplibregl = maplibregl
window.SkyVisorMap = { init, initMaps, initFleetFilmstrip, teardownMap, hasMap }
