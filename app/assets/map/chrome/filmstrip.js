import { reducedMotion } from '../../core/env.js'
import { AIRPORT_COORDS } from '../coords.js'

// Dashboard fleet filmstrip: clicking an airport chip flies the fleet map to it.
// Lives in the map bundle because it only matters on pages that render a map,
// and because it shares the coordinate table with fleet mode.
export function initFleetFilmstrip(root = document) {
  root.querySelectorAll('[data-fleet-filmstrip]').forEach((strip) => {
    if (strip.dataset.fleetFilmstripReady === 'true') return
    strip.dataset.fleetFilmstripReady = 'true'
    const mapEl = strip.parentElement?.querySelector('[data-map-mode="fleet"]')
    strip.querySelectorAll('[data-fleet-focus]').forEach((button) => {
      button.addEventListener('click', () => {
        strip.querySelectorAll('[data-fleet-focus]').forEach((item) => {
          item.classList.remove('border-primary/50', 'bg-primary/5')
        })
        button.classList.add('border-primary/50', 'bg-primary/5')
        const iata = button.dataset.fleetIata
        const coord = iata && AIRPORT_COORDS[iata]
        const map = mapEl?._skyvisorMap
        if (coord && map) {
          map.flyTo({ center: coord, zoom: 5.5, duration: reducedMotion() ? 0 : 900 })
        }
      })
    })
  })
}
