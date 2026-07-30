export const themeKey = 'skyvisor-theme'

// Applies a theme mode ('system' | 'light' | 'dark') and announces it.
// The skyvisor:theme event is the hook other subsystems (notably the map
// palette) use to repaint without re-reading localStorage.
export function applyTheme(mode) {
  const dark = mode === 'dark' || (mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.dataset.theme = mode
  document.documentElement.classList.toggle('dark', dark)
  localStorage.setItem(themeKey, mode)
  window.dispatchEvent(new CustomEvent('skyvisor:theme', { detail: { mode, dark } }))
}

export function currentTheme() {
  return localStorage.getItem(themeKey) || 'system'
}

// Keeps 'system' mode in sync when the OS preference flips mid-session.
export function initThemeWatcher() {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (currentTheme() === 'system') applyTheme('system')
  })
}
