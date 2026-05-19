import { useState } from 'react'

const API = window.location.hostname === 'localhost' ? 'http://localhost:8080' : 'https://showup-backend.fly.dev'

export default function CreatePage() {
  const [title, setTitle] = useState('')
  const [location, setLocation] = useState('')
  const [date, setDate] = useState('')
  const [time, setTime] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [slug, setSlug] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const eventUrl = slug ? `${window.location.origin}/events/${slug}` : ''

  async function handleCreate() {
    if (!title.trim()) return
    setSubmitting(true)
    try {
      const eventDate = date && time ? `${date}T${time}` : date || undefined
      const res = await fetch(`${API}/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: title.trim(),
          location: location.trim() || undefined,
          event_date: eventDate,
        }),
      })
      if (!res.ok) throw new Error('Failed to create event')
      const event = await res.json()
      setSlug(event.slug)
    } catch {
      alert('Something went wrong. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  function handleCopy() {
    navigator.clipboard.writeText(eventUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const inputClass = 'w-full bg-bg-surface rounded-[--radius-md] px-4 py-3 text-text-primary placeholder-text-muted focus:outline-none focus:ring-1 focus:ring-border-focus transition-colors'

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center px-4 py-12">
      <div className="w-full max-w-sm flex flex-col gap-6">

        <div className={`flex flex-col gap-1 transition-opacity ${slug ? 'opacity-40' : ''}`}>
          <h1 className="text-2xl font-bold text-text-primary">Create Event</h1>
          <p className="text-sm text-text-secondary">Fill in the details below, get a shareable link.</p>
        </div>

        <div className={`flex flex-col gap-4 transition-opacity ${slug ? 'opacity-30 pointer-events-none' : ''}`}>
          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-text-secondary">Event Name</label>
            <input
              type="text"
              placeholder="Pickup Volleyball"
              value={title}
              onChange={e => setTitle(e.target.value)}
              className={inputClass}
            />
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-text-secondary">Date</label>
            <input
              type="date"
              value={date}
              onChange={e => setDate(e.target.value)}
              className={inputClass}
            />
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-text-secondary">Time</label>
            <input
              type="time"
              value={time}
              onChange={e => setTime(e.target.value)}
              className={inputClass}
            />
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-text-secondary">Location</label>
            <input
              type="text"
              placeholder="Venice Beach Court 4"
              value={location}
              onChange={e => setLocation(e.target.value)}
              className={inputClass}
            />
          </div>
        </div>

        <button
          onClick={slug ? () => setSlug(null) : handleCreate}
          disabled={!slug && (!title.trim() || submitting)}
          className={`w-full py-3 rounded-[--radius-md] font-semibold text-sm transition-colors ${
            slug || title.trim()
              ? 'bg-accent text-white hover:bg-accent-secondary'
              : 'bg-bg-elevated text-text-disabled cursor-not-allowed'
          }`}
        >
          {slug ? 'Edit Event Details' : submitting ? 'Creating...' : 'Create Event →'}
        </button>

        {slug && (
          <div className="flex flex-col gap-3">
            <h2 className="text-xl font-bold text-text-primary">Share this link</h2>
            <div
              onClick={handleCopy}
              className="bg-bg-surface rounded-[--radius-lg] px-4 py-4 flex flex-col gap-1.5 cursor-pointer active:opacity-70 transition-opacity"
            >
              <p className="text-xs text-text-muted">Your shareable link</p>
              <p className="text-sm font-medium text-accent truncate">{eventUrl}</p>
              <p className="text-xs text-text-muted mt-1">
                {copied ? '✓ Copied!' : '⎘ tap to copy'}
              </p>
            </div>

            <a
              href={`/events/${slug}`}
              className="text-center text-sm text-text-muted hover:text-text-secondary transition-colors"
            >
              Preview your event page →
            </a>
          </div>
        )}

      </div>
    </div>
  )
}
