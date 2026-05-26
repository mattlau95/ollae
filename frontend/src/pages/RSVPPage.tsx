import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

const API = window.location.hostname === 'localhost' ? 'http://localhost:8080' : 'https://ollae-backend.fly.dev'

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
        document.title = `${data.event.title} · ollae.app`
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
  const canSubmit = !!(name.trim() && status)

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center px-4 py-6">
      <div className="w-full max-w-sm flex flex-col gap-6 flex-1">

        <Link to="/create">
          <img src="/ollae-logo.svg" alt="ollae" className="h-7 w-auto" />
        </Link>

        {/* Event info */}
        <div className="flex flex-col gap-2">
          <h1 className="text-[28px] font-bold text-text-primary leading-tight">
            🏐 {event!.title}
          </h1>
          <div className="flex flex-col gap-2 mt-1">
            {event!.event_date && (
              <p className="text-[13px] font-bold text-text-muted">📅  {formatDate(event!.event_date)}</p>
            )}
            {event!.event_date && hasTime(event!.event_date) && (
              <p className="text-[13px] font-bold text-text-muted">⏰  {formatTime(event!.event_date)}</p>
            )}
            {event!.location && (
              <p className="text-[13px] font-bold text-text-muted">📍 {event!.location}</p>
            )}
          </div>
        </div>

        {/* Attending count + activity feed */}
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-1.5">
            <span className="text-2xl font-bold text-text-primary">{attendingCount}</span>
            <span className="text-2xl font-normal text-text-primary">attending</span>
          </div>

          {responses.length > 0 && (
            <>
              <div className="bg-bg-surface rounded-lg overflow-hidden">
                <div className="px-4 pt-4 flex flex-col">
                  {visible.map(r => (
                    <p key={r.id} className="text-base text-text-muted leading-[150%] pb-4">
                      <span className="text-text-primary font-medium">{r.name}</span>
                      {' '}{statusText(r.status)}{' · '}{timeAgo(r.created_at)}
                    </p>
                  ))}
                </div>
              </div>
              {responses.length > 8 && (
                <div className="flex justify-center">
                  <button
                    onClick={() => setShowAll(v => !v)}
                    className="text-base text-text-primary underline underline-offset-2 transition-opacity hover:opacity-70"
                  >
                    {showAll ? 'Show less' : 'See All'}
                  </button>
                </div>
              )}
            </>
          )}
        </div>

        {/* Spacer — pushes RSVP form toward bottom on mobile, capped on desktop */}
        <div className="flex-1 max-h-12" />

        {/* RSVP form */}
        <div className="flex flex-col gap-6">
          <div className="flex flex-col gap-6">
            <h2 className="text-2xl font-semibold text-text-primary">RSVP</h2>

            {/* Name input */}
            <div className="flex flex-col gap-3">
              <input
                type="text"
                placeholder="Enter name here"
                value={name}
                onChange={e => setName(e.target.value)}
                className="w-full bg-bg-surface rounded-xl p-4 text-base text-text-primary placeholder-text-muted border border-white/[0.08] focus:outline-none focus:border-white/25 transition-colors"
              />
              <p className="text-sm text-text-muted text-center">
                Use a name the organizer would recognize as you.
              </p>
            </div>

            {/* RSVP buttons */}
            <div className="flex gap-4">
              {/* I'm in */}
              <button
                onClick={() => setStatus('in')}
                className={`flex-1 flex flex-col items-center justify-center gap-4 py-5 px-8 rounded-xl border transition-colors ${
                  status === 'in'
                    ? 'bg-[#DCFCE7] border-[#16A34A]'
                    : 'bg-bg-elevated border-white/[0.08]'
                }`}
              >
                <span className="text-[32px] leading-none">✅</span>
                <span className={`text-xl font-normal ${status === 'in' ? 'text-[#22C55E]' : 'text-text-primary'}`}>
                  I'm in
                </span>
              </button>

              {/* Can't make it */}
              <button
                onClick={() => setStatus('out')}
                className={`flex-1 flex flex-col items-center justify-center gap-4 py-5 px-8 rounded-xl border transition-colors ${
                  status === 'out'
                    ? 'bg-[#FEE2E2] border-[#DC2626]'
                    : 'bg-bg-elevated border-white/[0.08]'
                }`}
              >
                <span className={`text-[32px] leading-none ${status === 'out' ? '' : 'opacity-60'}`}>😔</span>
                <span className={`text-xl font-normal ${status === 'out' ? 'text-[#EF4444]' : 'text-text-primary'}`}>
                  Can't make it
                </span>
              </button>
            </div>
          </div>

          {/* Not sure yet */}
          <div className="flex justify-center">
            <button
              onClick={() => setStatus('remind_me')}
              className={`text-base underline underline-offset-2 transition-opacity hover:opacity-70 ${
                status === 'remind_me' ? 'text-status-remind' : 'text-text-primary'
              }`}
            >
              Not sure yet — remind me closer to the date
            </button>
          </div>

          {/* Submit */}
          <button
            onClick={handleSubmit}
            disabled={!canSubmit || submitting}
            className={`w-full py-4 rounded-xl text-xl font-normal border transition-colors ${
              canSubmit
                ? 'bg-[#F59E0B] text-[#F8FAFC] border-[#F8FAFC] hover:opacity-90'
                : 'bg-bg-surface text-text-muted border-white/[0.08] cursor-not-allowed'
            }`}
          >
            {submitting ? 'Submitting...' : 'Submit ->'}
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
  const statusColor = status === 'in' ? 'text-[#22C55E]' : status === 'out' ? 'text-[#EF4444]' : 'text-status-remind'

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center justify-center px-4 gap-6 text-center">
      <div className="text-[64px] leading-none">{emoji}</div>
      <h2 className="text-2xl font-semibold text-text-primary">{message}</h2>
      <p className={`text-base ${statusColor}`}>{statusLine}</p>
      <p className="text-base text-text-disabled max-w-xs">
        Changed your plans? Just reopen this link and resubmit your name.
      </p>
      <button onClick={onBack} className="text-base text-text-primary underline underline-offset-2 transition-opacity hover:opacity-70">
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
