import { useState } from 'react'
import { Copy, Check } from 'lucide-react'

const API = 'http://localhost:8080'

export default function CreatePage() {
  const [title, setTitle] = useState('')
  const [location, setLocation] = useState('')
  const [date, setDate] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [slug, setSlug] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const eventUrl = slug ? `${window.location.origin}/events/${slug}` : ''

  async function handleCreate() {
    if (!title.trim()) return
    setSubmitting(true)
    try {
      const res = await fetch(`${API}/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: title.trim(),
          location: location.trim() || undefined,
          event_date: date || undefined,
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

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center px-4 py-12">
      <div className="w-full max-w-sm flex flex-col gap-8">

        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold text-text-primary">Create Event</h1>
          <p className="text-sm text-text-secondary">Fill in the details below, get a shareable link.</p>
        </div>

        <div className={`flex flex-col gap-4 transition-opacity ${slug ? 'opacity-30 pointer-events-none' : ''}`}>
          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-text-secondary uppercase tracking-wide">Event Name *</label>
            <input
              type="text"
              placeholder="Weekly Pickup Volleyball"
              value={title}
              onChange={e => setTitle(e.target.value)}
              className="w-full bg-bg-surface border border-border rounded-[--radius-md] px-4 py-3 text-text-primary placeholder-text-muted focus:outline-none focus:border-border-focus transition-colors"
            />
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-text-secondary uppercase tracking-wide">Date</label>
            <input
              type="datetime-local"
              value={date}
              onChange={e => setDate(e.target.value)}
              className="w-full bg-bg-surface border border-border rounded-[--radius-md] px-4 py-3 text-text-primary focus:outline-none focus:border-border-focus transition-colors"
            />
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-xs font-medium text-text-secondary uppercase tracking-wide">Location</label>
            <input
              type="text"
              placeholder="Sunset Park, Court 3"
              value={location}
              onChange={e => setLocation(e.target.value)}
              className="w-full bg-bg-surface border border-border rounded-[--radius-md] px-4 py-3 text-text-primary placeholder-text-muted focus:outline-none focus:border-border-focus transition-colors"
            />
          </div>

          <button
            onClick={handleCreate}
            disabled={!title.trim() || submitting}
            className={`w-full py-3 rounded-[--radius-md] font-semibold text-sm transition-colors ${
              title.trim()
                ? 'bg-accent text-bg-base hover:bg-accent-secondary'
                : 'bg-bg-elevated text-text-disabled cursor-not-allowed'
            }`}
          >
            {slug ? 'Edit Event Details' : submitting ? 'Creating...' : 'Create Event →'}
          </button>
        </div>

        {slug && (
          <div className="flex flex-col gap-3">
            <p className="text-sm font-medium text-text-secondary uppercase tracking-wide">Share this link</p>
            <div className="flex items-center gap-2 bg-bg-surface border border-border rounded-[--radius-md] px-4 py-3">
              <span className="text-sm text-text-primary flex-1 truncate">{eventUrl}</span>
              <button onClick={handleCopy} className="text-text-muted hover:text-accent transition-colors flex-shrink-0">
                {copied ? <Check size={16} className="text-status-in" /> : <Copy size={16} />}
              </button>
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
