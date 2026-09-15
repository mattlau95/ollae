import confetti from 'canvas-confetti'

export function celebrate() {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  confetti({ particleCount: 80, spread: 55, origin: { y: 0.5 }, ticks: 180 })
}
