import { reducedMotion } from '../../core/env.js'

// Dashboard fleet filmstrip: clicking an airport chip flies the fleet map to it.
// Coordinates come from data attributes resolved server-side; a watch whose
// origin could not be resolved simply does not move the map.
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

        const lon = Number(button.dataset.fleetLon)
        const lat = Number(button.dataset.fleetLat)
        const map = mapEl?._skyvisorMap
        if (!map || !Number.isFinite(lon) || !Number.isFinite(lat)) return
        if (button.dataset.fleetLon === '' || button.dataset.fleetLat === '') return

        map.flyTo({ center: [lon, lat], zoom: 5.5, duration: reducedMotion() ? 0 : 900 })
      })
    })
  })
}
