import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'

const API = window.location.hostname === 'localhost' ? 'http://localhost:8080' : 'https://showup-backend.fly.dev'

type Response = {
  id: string
  name: string
  status: 'in' | 'out' | 'remind_me'
  created_at: string
}

type Event = {
  id: string
  slug: string
  title: string
  location: string
  event_date: string | null
  created_at: string
}

type RSVPStatus = 'in' | 'out' | 'remind_me' | null

export default function RSVPPage() {
  const { slug } = useParams<{ slug: string }>()

  const [event, setEvent] = useState<Event | null>(null)
  const [responses, setResponses] = useState<Response[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAll, setShowAll] = useState(false)

  const [name, setName] = useState('')
  const [status, setStatus] = useState<RSVPStatus>(null)
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(false)

  useEffect(() => {
    fetch(`${API}/events/${slug}`)
      .then(r => {
        if (!r.ok) throw new Error('Event not found')
        return r.json()
      })
      .then(data => {
        setEvent(data.event)
        setResponses(data.responses)
        setLoading(false)
      })
      .catch(e => {
        setError(e.message)
        setLoading(false)
      })
  }, [slug])

  async function handleSubmit() {
    if (!name.trim() || !status) return
    setSubmitting(true)
    try {
      const res = await fetch(`${API}/events/${slug}/rsvp`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name.trim(), status }),
      })
      if (!res.ok) throw new Error('Failed to submit')
      const updated: Response[] = await res.json()
      setResponses(updated)
      setSubmitted(true)
    } catch {
      alert('Something went wrong. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return (
    <div className="min-h-screen bg-bg-base flex items-center justify-center text-text-muted">
      Loading...
    </div>
  )

  if (error) return (
    <div className="min-h-screen bg-bg-base flex items-center justify-center text-text-muted">
      {error}
    </div>
  )

  if (submitted) return <SuccessScreen name={name} status={status!} onBack={() => setSubmitted(false)} />

  const attendingCount = responses.filter(r => r.status === 'in').length
  const sorted = [...responses].reverse()
  const visible = showAll ? sorted : sorted.slice(0, 8)

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center px-4 py-8">
      <div className="w-full max-w-sm flex flex-col gap-6">

        {/* Event info */}
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold text-text-primary">🏐 {event!.title}</h1>
          <div className="flex flex-col gap-0.5 mt-1">
            {event!.event_date && (
              <p className="text-sm font-medium text-text-secondary">📅 {formatDate(event!.event_date)}</p>
            )}
            {event!.event_date && hasTime(event!.event_date) && (
              <p className="text-sm font-medium text-text-secondary">⏰ {formatTime(event!.event_date)}</p>
            )}
            {event!.location && (
              <p className="text-sm font-medium text-text-secondary">📍 {event!.location}</p>
            )}
          </div>
        </div>

        {/* Attending count */}
        <div className="flex items-baseline gap-1.5">
          <span className="text-4xl font-bold text-text-primary">{attendingCount}</span>
          <span className="text-lg text-text-secondary">attending</span>
        </div>

        {/* Activity feed */}
        {responses.length > 0 && (
          <div className="flex flex-col gap-2">
            <div className="bg-bg-surface rounded-[--radius-lg] px-4 py-3 flex flex-col gap-2">
              {visible.map(r => (
                <p key={r.id} className="text-sm text-text-secondary">
                  <span className="text-text-primary font-medium">{r.name}</span>
                  {' '}{statusText(r.status)}{' · '}{timeAgo(r.created_at)}
                </p>
              ))}
            </div>
            {responses.length > 8 && (
              <button
                onClick={() => setShowAll(v => !v)}
                className="text-sm text-text-muted hover:text-text-secondary transition-colors self-start underline underline-offset-2"
              >
                {showAll ? 'Show less' : 'See All'}
              </button>
            )}
          </div>
        )}

        {/* RSVP form */}
        <div className="flex flex-col gap-4">
          <h2 className="text-xl font-bold text-text-primary">RSVP</h2>

          <div className="flex flex-col gap-1.5">
            <input
              type="text"
              placeholder="Enter name here"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full bg-bg-surface rounded-[--radius-md] px-4 py-3 text-text-primary placeholder-text-muted focus:outline-none focus:ring-1 focus:ring-border-focus transition-colors"
            />
            <p className="text-xs text-text-muted px-1">Use a name the organizer would recognize as you.</p>
          </div>

          <div className="flex gap-3">
            <button
              onClick={() => setStatus('in')}
              className={`flex-1 py-4 rounded-[--radius-md] font-medium text-sm transition-colors flex flex-col items-center gap-2 ${
                status === 'in'
                  ? 'bg-green-100 border border-green-600 text-green-600'
                  : 'bg-bg-surface text-text-primary'
              }`}
            >
              <span className="text-2xl">✅</span>
              I'm in
            </button>
            <button
              onClick={() => setStatus('out')}
              className={`flex-1 py-4 rounded-[--radius-md] font-medium text-sm transition-colors flex flex-col items-center gap-2 ${
                status === 'out'
                  ? 'bg-red-100 border border-red-600 text-red-600'
                  : 'bg-bg-surface text-text-primary'
              }`}
            >
              <span className="text-2xl">😔</span>
              Can't make it
            </button>
          </div>

          <button
            onClick={() => setStatus('remind_me')}
            className="text-center text-sm text-text-muted hover:text-text-secondary transition-colors underline underline-offset-2"
          >
            Not sure yet — remind me closer to the date
          </button>

          <button
            onClick={handleSubmit}
            disabled={!name.trim() || !status || submitting}
            className={`w-full py-3 rounded-[--radius-md] font-semibold text-sm transition-colors ${
              name.trim() && status
                ? 'bg-accent text-white hover:bg-accent-secondary'
                : 'bg-bg-elevated text-text-disabled cursor-not-allowed'
            }`}
          >
            {submitting ? 'Submitting...' : 'Submit →'}
          </button>
        </div>

      </div>
    </div>
  )
}

function SuccessScreen({ name, status, onBack }: { name: string; status: RSVPStatus; onBack: () => void }) {
  const emoji = status === 'in' ? '🙌' : status === 'out' ? '😢' : '🔔'
  const message = status === 'in' ? "You're on the list!" : status === 'out' ? "Got it, you're out." : "We'll remind you!"
  const statusLine = status === 'in' ? `${name} · I'm in 🏐` : status === 'out' ? `${name} · Can't make it 😔` : `${name} · Remind me 🔔`
  const statusColor = status === 'in' ? 'text-status-in' : status === 'out' ? 'text-status-out' : 'text-status-remind'

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center justify-center px-4 gap-4 text-center">
      <div className="text-5xl">{emoji}</div>
      <h2 className="text-2xl font-bold text-text-primary">{message}</h2>
      <p className={`text-sm font-medium ${statusColor}`}>{statusLine}</p>
      <p className="text-sm text-text-muted max-w-xs">
        Changed your plans? Just reopen this link and resubmit your name.
      </p>
      <button onClick={onBack} className="text-sm text-text-muted hover:text-text-secondary transition-colors mt-2 underline underline-offset-2">
        ← View who's coming
      </button>
    </div>
  )
}

function statusText(status: string) {
  const map: Record<string, string> = {
    in: 'is in 🏐',
    out: "can't make it 😔",
    remind_me: 'wants a reminder 🔔',
  }
  return map[status] ?? status
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', {
    weekday: 'long', month: 'long', day: 'numeric',
  })
}

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleTimeString('en-US', {
    hour: 'numeric', minute: '2-digit', hour12: true,
  })
}

function hasTime(dateStr: string) {
  const d = new Date(dateStr)
  return d.getHours() !== 0 || d.getMinutes() !== 0
}

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.floor(hrs / 24)}d ago`
}
