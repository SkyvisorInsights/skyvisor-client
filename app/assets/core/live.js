// Singleton, ref-counted SSE connection to /events.
//
// Previously every [data-live-refresh-url] element opened its own EventSource.
// A page with a stats rail, a drawer and a map would hold three connections
// against the browser's ~6-per-origin HTTP/1.1 cap, on a route that keeps each
// connection open indefinitely. One shared connection fans out instead.

const ENDPOINT = '/events'

let source = null
let refs = 0
let offline = false

const handlers = new Map() // eventName -> Set<fn>
const bound = new Map() // eventName -> native listener currently attached to `source`

function dispatch(name, event) {
  const subscribers = handlers.get(name)
  if (!subscribers || !subscribers.size) return

  let payload = null
  if (event.data) {
    try {
      payload = JSON.parse(event.data)
    } catch {
      payload = null // non-JSON frames (keepalives) still notify subscribers
    }
  }

  subscribers.forEach((fn) => {
    try {
      fn(payload, event)
    } catch (error) {
      console.error('[skyvisor] live subscriber failed', error)
    }
  })
}

function bind(name) {
  if (!source || bound.has(name)) return
  const listener = (event) => dispatch(name, event)
  bound.set(name, listener)
  source.addEventListener(name, listener)
}

function teardown() {
  if (!source) return
  bound.forEach((listener, name) => source.removeEventListener(name, listener))
  bound.clear()
  source.removeEventListener('error', onError)
  source.close()
  source = null
}

function onError() {
  if (!source) return
  // CONNECTING means the browser is already retrying on its own schedule.
  // CLOSED means it gave up — which is what happens when the server rejects the
  // request outright (an expired session returning 401, say). Reopening here
  // would hammer the endpoint, so stop and let the page decide what to show.
  if (source.readyState !== EventSource.CLOSED) return
  teardown()
  offline = true
  window.dispatchEvent(new CustomEvent('skyvisor:live-offline'))
}

function open() {
  if (source || offline || typeof EventSource === 'undefined') return
  source = new EventSource(ENDPOINT)
  source.addEventListener('error', onError)
  handlers.forEach((_, name) => bind(name))
}

// Subscribes fn to one or more named SSE events. Returns a release function;
// the shared connection closes once every subscriber has released.
export function subscribe(events, fn) {
  const names = Array.isArray(events) ? events : [events]

  names.forEach((name) => {
    if (!handlers.has(name)) handlers.set(name, new Set())
    handlers.get(name).add(fn)
  })

  refs += 1
  open()
  names.forEach(bind)

  let released = false
  return () => {
    if (released) return
    released = true

    names.forEach((name) => {
      const subscribers = handlers.get(name)
      if (!subscribers) return
      subscribers.delete(fn)
      if (subscribers.size) return
      handlers.delete(name)
      const listener = bound.get(name)
      if (listener && source) source.removeEventListener(name, listener)
      bound.delete(name)
    })

    refs = Math.max(0, refs - 1)
    if (refs === 0) teardown()
  }
}

// Clears the offline latch so a reconnect can be attempted (e.g. after re-auth).
export function resumeLive() {
  offline = false
  if (refs > 0) open()
}

export function liveStatus() {
  return { connected: Boolean(source), refs, offline }
}

const REFRESH_EVENTS = ['flight.updated', 'flight.delayed', 'flight.cancelled', 'gate.changed']
const REFRESH_DEBOUNCE_MS = 450

// Wires elements that want to re-fetch themselves over htmx when live data moves.
export function initLiveRefresh(root = document) {
  if (typeof EventSource === 'undefined') return

  root.querySelectorAll('[data-live-refresh-url]').forEach((element) => {
    if (element.dataset.liveRefreshReady === 'true') return
    element.dataset.liveRefreshReady = 'true'

    let timer
    const refresh = () => {
      window.clearTimeout(timer)
      timer = window.setTimeout(() => {
        if (!document.documentElement.contains(element)) return
        window.htmx.ajax('GET', element.dataset.liveRefreshUrl, {
          target: element.dataset.liveRefreshTarget,
          swap: 'outerHTML',
        })
      }, REFRESH_DEBOUNCE_MS)
    }

    const release = subscribe(REFRESH_EVENTS, refresh)
    element._skyvisorLiveRelease = () => {
      window.clearTimeout(timer)
      release()
    }
  })
}

// Called from htmx:beforeCleanupElement so a swapped-away element drops its
// subscription instead of leaking a reference to a detached node.
export function releaseLiveElement(element) {
  const release = element && element._skyvisorLiveRelease
  if (!release) return
  release()
  element._skyvisorLiveRelease = null
}
