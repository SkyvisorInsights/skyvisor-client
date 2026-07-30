import { animate, stagger } from 'motion'
import { reducedMotion } from './env.js'

function animateCounter(element) {
  const target = Number(element.dataset.motionCounter || element.textContent.trim())
  if (!Number.isFinite(target)) return
  if (reducedMotion()) {
    element.textContent = String(Math.round(target))
    return
  }
  animate(0, target, {
    duration: 0.75,
    ease: 'easeOut',
    onUpdate: (value) => {
      element.textContent = String(Math.round(value))
    },
  })
}

function animateProgressBars(root) {
  root.querySelectorAll('[data-motion-progress]').forEach((element) => {
    const target = Number(element.dataset.motionProgress || 0)
    const bar = element.querySelector('.flight-progress-bar')
    const plane = element.querySelector('.flight-progress-plane')
    if (!bar || !plane) return
    if (reducedMotion()) {
      bar.style.width = `${target}%`
      plane.style.left = `calc(${target}% - 5px)`
      return
    }
    animate(0, target, {
      duration: 0.9,
      ease: 'easeOut',
      onUpdate: (value) => {
        bar.style.width = `${value}%`
        plane.style.left = `calc(${value}% - 5px)`
      },
    })
  })
}

function animateEnter(root) {
  if (reducedMotion()) return
  const items = root.querySelectorAll('[data-motion-enter]')
  if (!items.length) return
  animate(
    items,
    { opacity: [0, 1], y: [14, 0] },
    { duration: 0.45, delay: stagger(0.06), ease: 'easeOut' },
  )
}

function pulseRiskMetrics(root) {
  root.querySelectorAll('[data-motion-pulse]').forEach((element) => {
    if (reducedMotion()) return
    animate(element, { scale: [1, 1.04, 1] }, { duration: 1.2, ease: 'easeInOut' })
  })
}

export function initMotion(root = document) {
  root.querySelectorAll('[data-motion-counter]').forEach((element) => {
    if (element.dataset.motionCounterReady === 'true') return
    element.dataset.motionCounterReady = 'true'
    animateCounter(element)
  })
  animateEnter(root)
  animateProgressBars(root)
  pulseRiskMetrics(root)
}
