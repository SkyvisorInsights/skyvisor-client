// WebGL capability probe.
//
// The server renders a real globe as inline SVG, so a device without WebGL is
// not left with a blank box — it simply keeps the fallback. This decides
// whether to attempt a MapLibre instance at all, so a failed context creation
// never removes a working picture.

let cached = null

export function hasWebGL() {
  if (cached !== null) return cached
  try {
    const canvas = document.createElement('canvas')
    const context = canvas.getContext('webgl2') || canvas.getContext('webgl')
    cached = Boolean(context)
    // Release the probe context immediately; browsers cap how many exist.
    const lose = context && context.getExtension && context.getExtension('WEBGL_lose_context')
    if (lose) lose.loseContext()
  } catch {
    cached = false
  }
  return cached
}

// Rough capability tier, used to drop the more expensive layers on weak
// hardware rather than shipping a globe that stutters.
export function isLowPowerDevice() {
  const cores = navigator.hardwareConcurrency
  if (typeof cores === 'number' && cores > 0 && cores <= 4) return true
  return window.matchMedia('(pointer: coarse)').matches
}
