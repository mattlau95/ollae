import { useLayoutEffect, useRef } from 'react'

// The portfolio case study frames two screens with ?embed=1. These are the
// pages allowed to frame them; vercel.json and the backend's framing.go
// send the same list as CSP frame-ancestors.
export const EMBED_PARENTS = [
  'https://www.matthewclau.com',
  'https://matthewclau.com',
  'http://localhost:4321',
]

// The permanent guestbook event. The backend and vercel.json name it too.
export const GUESTBOOK_SLUG = 'wssrfd7v'

export const isEmbed = new URLSearchParams(window.location.search).get('embed') === '1'

const framed = window.parent !== window

// Links that leave a framed screen open in a new tab instead of navigating
// the frame.
export const newTab = isEmbed ? { target: '_blank', rel: 'noopener noreferrer' } : {}

function originOf(url: string | undefined) {
  if (!url) return null
  try {
    return new URL(url).origin
  } catch {
    return null
  }
}

let parentOrigin: string | null = null

// The parent's origin, only if it's on the allowlist. Messages are posted to
// that exact origin, never '*'.
//
// 1. location.ancestorOrigins (Chromium, WebKit) names the parent directly.
// 2. Firefox lacks it, so fall back to document.referrer. That's the parent
//    page only because the embed URL loads the app directly
//    (/events/:slug?_src=app&embed=1); through the backend's redirect page
//    the referrer would be ollae.app.
// 3. A message from an allowlisted parent (lockParentOrigin) also counts.
//
// If none of these gives an allowlisted origin, nothing is posted.
function resolveParentOrigin() {
  if (parentOrigin || !framed) return parentOrigin
  const candidates = [window.location.ancestorOrigins?.[0], originOf(document.referrer)]
  parentOrigin = candidates.find(o => o && EMBED_PARENTS.includes(o)) ?? null
  return parentOrigin
}

export function lockParentOrigin(origin: string) {
  if (framed && EMBED_PARENTS.includes(origin)) parentOrigin = origin
}

export type EmbedMessage =
  | { type: 'ollae:height'; height: number }
  | { type: 'ollae:ready' }
  | { type: 'ollae:input'; text: string }
  | { type: 'ollae:created' }

export function postToParent(message: EmbedMessage) {
  if (!isEmbed) return
  const origin = resolveParentOrigin()
  if (origin) window.parent.postMessage(message, origin)
}

// Reports the content wrapper's height to the parent whenever it changes.
// It observes the wrapper, not the document, and embed mode drops
// min-h-screen, so resizing the frame can't change what's measured.
//
// When screenKey changes (RSVP form → success screen), the new screen starts
// at least as tall as the last one so the portfolio page doesn't jump.
export function useEmbedHeight<T extends HTMLElement>(screenKey: string) {
  const ref = useRef<T>(null)
  const lastHeight = useRef(0)

  useLayoutEffect(() => {
    const el = ref.current
    if (!isEmbed || !el) return
    if (lastHeight.current) el.style.minHeight = `${lastHeight.current}px`

    const observer = new ResizeObserver(() => {
      const height = Math.ceil(el.getBoundingClientRect().height)
      if (height === lastHeight.current) return
      lastHeight.current = height
      postToParent({ type: 'ollae:height', height })
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [screenKey])

  return ref
}

// Sets <meta name="robots" content="noindex"> while mounted.
export function setNoindex() {
  let el = document.querySelector('meta[name="robots"]')
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('name', 'robots')
    document.head.appendChild(el)
  }
  el.setAttribute('content', 'noindex')
}
