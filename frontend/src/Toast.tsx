import { useEffect, useRef } from 'react'

type Props = { message: string | null; onDismiss: () => void }

export function Toast({ message, onDismiss }: Props) {
  const dismissRef = useRef(onDismiss)
  useEffect(() => {
    dismissRef.current = onDismiss
  })

  useEffect(() => {
    if (!message) return
    const id = setTimeout(() => dismissRef.current(), 5000)
    return () => clearTimeout(id)
  }, [message])

  if (!message) return null
  return (
    <div
      role="alert"
      className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 w-[calc(100%-2rem)] max-w-sm rounded-xl bg-bg-elevated border border-white/[0.12] px-4 py-3 text-sm text-text-primary shadow-lg flex items-start gap-3"
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
