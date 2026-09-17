import { useEffect, useRef } from 'react'

import { isEmbed } from './embed'

type Props = { message: string | null; onDismiss: () => void }

// In embed mode the frame is as tall as its content, so a toast fixed to the
// bottom of the frame could be far below what the visitor can see. There it
// renders in place, where the page puts it.
export function Toast({ message, onDismiss }: Props) {
  const dismissRef = useRef(onDismiss)
  useEffect(() => {
    dismissRef.current = onDismiss
  })

  useEffect(() => {
    if (!message) return
    const id = setTimeout(() => dismissRef.current(), isEmbed ? 8000 : 5000)
    return () => clearTimeout(id)
  }, [message])

  if (!message) return null
  const position = isEmbed ? 'w-full' : 'fixed bottom-6 left-1/2 -translate-x-1/2 z-50 w-[calc(100%-2rem)] max-w-sm shadow-lg'
  return (
    <div
      role="alert"
      className={`${position} rounded-xl bg-bg-elevated border border-white/[0.12] px-4 py-3 text-sm text-text-primary flex items-start gap-3`}
    >
      <span className="flex-1">{message}</span>
      <button
        onClick={onDismiss}
        aria-label="Dismiss"
        className="text-text-muted hover:text-text-primary w-6 h-6 -mr-1 -mt-0.5 inline-flex items-center justify-center shrink-0"
      >
        ✕
      </button>
    </div>
  )
}
