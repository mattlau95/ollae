import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Bell, MapPin, Calendar, Users } from 'lucide-react'

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

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center px-4 py-8">
      <div className="w-full max-w-sm flex flex-col gap-6">

        {/* Event info */}
        <div className="flex flex-col gap-2">
          <h1 className="text-2xl font-semibold text-text-primary">{event!.title}</h1>
          <div className="flex flex-col gap-1">
            {event!.event_date && (
              <div className="flex items-center gap-2 text-sm text-text-secondary">
                <Calendar size={14} />
                {new Date(event!.event_date).toLocaleDateString('en-US', {
                  weekday: 'long', month: 'long', day: 'numeric',
                })}
              </div>
            )}
            {event!.location && (
              <div className="flex items-center gap-2 text-sm text-text-secondary">
                <MapPin size={14} />
                {event!.location}
              </div>
            )}
          </div>
        </div>

        {/* Attending count */}
        <div className="flex items-center gap-2 text-text-secondary text-sm">
          <Users size={14} />
          <span>
            <span className="text-text-primary font-semibold text-lg">{attendingCount}</span>
            {' '}attending
          </span>
        </div>

        {/* Activity feed */}
        {responses.length > 0 && (
          <div className="bg-bg-surface rounded-[--radius-lg] p-4 flex flex-col gap-3">
            {responses.slice(-8).reverse().map(r => (
              <div key={r.id} className="flex items-center gap-2 text-sm">
                <span className={statusDot(r.status)} />
                <span className="text-text-primary font-medium">{r.name}</span>
                <span className="text-text-muted">{statusLabel(r.status)}</span>
                <span className="text-text-disabled ml-auto">
                  {timeAgo(r.created_at)}
                </span>
              </div>
            ))}
          </div>
        )}

        {/* RSVP form */}
        <div className="flex flex-col gap-4">
          <input
            type="text"
            placeholder="Your name"
            value={name}
            onChange={e => setName(e.target.value)}
            className="w-full bg-bg-surface border border-border rounded-[--radius-md] px-4 py-3 text-text-primary placeholder-text-muted focus:outline-none focus:border-border-focus transition-colors"
          />

          <div className="flex gap-3">
            <button
              onClick={() => setStatus('in')}
              className={`flex-1 py-3 rounded-[--radius-md] font-medium text-sm transition-colors ${
                status === 'in'
                  ? 'bg-status-in-bg border border-status-in-border text-status-in'
                  : 'bg-bg-surface border border-border text-text-secondary hover:border-status-in-border hover:text-status-in'
              }`}
            >
              I'm in
            </button>
            <button
              onClick={() => setStatus('out')}
              className={`flex-1 py-3 rounded-[--radius-md] font-medium text-sm transition-colors ${
                status === 'out'
                  ? 'bg-status-out-bg border border-status-out-border text-status-out'
                  : 'bg-bg-surface border border-border text-text-secondary hover:border-status-out-border hover:text-status-out'
              }`}
            >
              Can't make it
            </button>
          </div>

          <button
            onClick={handleSubmit}
            disabled={!name.trim() || !status || submitting}
            className={`w-full py-3 rounded-[--radius-md] font-semibold text-sm transition-colors ${
              name.trim() && status
                ? 'bg-accent text-bg-base hover:bg-accent-secondary'
                : 'bg-bg-elevated text-text-disabled cursor-not-allowed'
            }`}
          >
            {submitting ? 'Submitting...' : 'Submit'}
          </button>

          <button
            onClick={() => setStatus('remind_me')}
            className="text-center text-sm text-text-muted hover:text-text-secondary transition-colors"
          >
            <Bell size={12} className="inline mr-1" />
            Not sure yet — remind me closer to the date
          </button>
        </div>

      </div>
    </div>
  )
}

function SuccessScreen({ name, status, onBack }: { name: string; status: RSVPStatus; onBack: () => void }) {
  const emoji = status === 'in' ? '🎉' : status === 'out' ? '😢' : '🔔'
  const message = status === 'in' ? "You're on the list!" : status === 'out' ? "Got it, you're out." : "We'll remind you!"

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center justify-center px-4 gap-4 text-center">
      <div className="text-5xl">{emoji}</div>
      <h2 className="text-2xl font-semibold text-text-primary">{message}</h2>
      <p className={`text-sm font-medium ${statusTextColor(status)}`}>
        {name} · {statusLabel(status)}
      </p>
      <p className="text-sm text-text-muted max-w-xs">
        Changed your plans? Just reopen this link and resubmit your name.
      </p>
      <button onClick={onBack} className="text-sm text-text-muted hover:text-text-secondary transition-colors mt-2">
        ← View who's coming
      </button>
    </div>
  )
}

function statusDot(status: string) {
  const map: Record<string, string> = {
    in: 'w-2 h-2 rounded-full bg-status-in flex-shrink-0',
    out: 'w-2 h-2 rounded-full bg-status-out flex-shrink-0',
    remind_me: 'w-2 h-2 rounded-full bg-status-remind flex-shrink-0',
  }
  return map[status] ?? 'w-2 h-2 rounded-full bg-text-muted flex-shrink-0'
}

function statusLabel(status: RSVPStatus | string) {
  const map: Record<string, string> = { in: 'is in', out: 'is out', remind_me: 'wants a reminder' }
  return map[status as string] ?? status
}

function statusTextColor(status: RSVPStatus) {
  const map: Record<string, string> = {
    in: 'text-status-in', out: 'text-status-out', remind_me: 'text-status-remind',
  }
  return map[status as string] ?? 'text-text-secondary'
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
