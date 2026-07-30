// Shared environment probes. Kept dependency-free so both the app and map
// bundles can import it without pulling anything else along.

export const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

export function parseNumber(value) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}
