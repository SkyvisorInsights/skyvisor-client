// Resolves the --map-* design tokens into colours MapLibre can consume.
//
// The tokens are authored in OKLCH so they sit in the same colour space as the
// rest of the design system. MapLibre's style parser understands neither
// oklch() nor CSS custom properties, so we hand the value to the browser's own
// colour engine (canvas fillStyle) and read back sRGB. That matches exactly how
// the same token renders elsewhere on the page, with no conversion maths and no
// dependency.

export const PALETTE_KEYS = [
  'ocean',
  'land',
  'boundary',
  'graticule',
  'arc-ok',
  'arc-watch',
  'arc-risk',
  'hub',
  'flight',
  'label',
  'label-halo',
  'atmosphere',
]

// Last-resort colours. If a token is renamed or the canvas trick is unavailable
// we still render a legible map rather than black on black.
export const PALETTE_FALLBACK = {
  ocean: '#0b1220',
  land: '#1c2536',
  boundary: '#33405a',
  graticule: '#212c40',
  'arc-ok': '#60a5fa',
  'arc-watch': '#f0a63a',
  'arc-risk': '#f2555a',
  hub: '#b9cdf5',
  flight: '#f4f7fe',
  label: '#e8ecf6',
  'label-halo': '#0b1220',
  atmosphere: '#4f7fe0',
}

let probe = null

function probeContext() {
  if (probe !== null) return probe
  try {
    const canvas = document.createElement('canvas')
    canvas.width = 1
    canvas.height = 1
    probe = canvas.getContext('2d', { willReadFrequently: true })
  } catch {
    probe = false
  }
  return probe
}

// Converts any CSS colour the browser understands into an rgba() string.
// Returns null when the input is not a colour the browser accepts.
export function cssColorToRGBA(value) {
  const ctx = probeContext()
  if (!ctx || !value) return null

  const input = String(value).trim()
  if (!input) return null

  // Assigning an invalid colour leaves fillStyle untouched, so seed it with a
  // sentinel and check whether the assignment actually took effect.
  ctx.fillStyle = '#000000'
  ctx.fillStyle = input
  const first = ctx.fillStyle
  ctx.fillStyle = '#ffffff'
  ctx.fillStyle = input
  if (ctx.fillStyle !== first) return null

  ctx.clearRect(0, 0, 1, 1)
  ctx.fillRect(0, 0, 1, 1)
  const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data
  return `rgba(${r}, ${g}, ${b}, ${Number((a / 255).toFixed(3))})`
}

// Reads the current --map-* tokens for the active theme.
export function resolveMapPalette(root = document.documentElement) {
  const styles = getComputedStyle(root)
  const palette = {}
  for (const key of PALETTE_KEYS) {
    const raw = styles.getPropertyValue(`--map-${key}`)
    palette[key] = cssColorToRGBA(raw) ?? PALETTE_FALLBACK[key]
  }
  return palette
}
