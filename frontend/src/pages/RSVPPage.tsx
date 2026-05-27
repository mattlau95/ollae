import { useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'

const API = window.location.hostname === 'localhost' ? 'http://localhost:8080' : 'https://ollae-backend.fly.dev'

type Response = {
  id: string
  name: string
  status: 'in' | 'out' | 'remind_me'
  guests: number
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
  const [searchParams] = useSearchParams()
  const adminToken = searchParams.get('admin')
  const isAdmin = !!adminToken

  const [event, setEvent] = useState<Event | null>(null)
  const [responses, setResponses] = useState<Response[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAll, setShowAll] = useState(false)

  const [name, setName] = useState('')
  const [status, setStatus] = useState<RSVPStatus>(null)
  const [guests, setGuests] = useState(0)
  const [showCustomGuests, setShowCustomGuests] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(false)

  function handleSetStatus(s: RSVPStatus) {
    setStatus(s)
    if (s !== 'in') {
      setGuests(0)
      setShowCustomGuests(false)
    }
  }

  // Edit state (admin only)
  const [isEditing, setIsEditing] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const [editDate, setEditDate] = useState('')
  const [editTime, setEditTime] = useState('')
  const [editLocation, setEditLocation] = useState('')
  const [saving, setSaving] = useState(false)

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

        // Pre-fill edit form fields
        setEditTitle(data.event.title)
        setEditLocation(data.event.location ?? '')
        if (data.event.event_date) {
          const stripped = data.event.event_date.replace(/Z$|[+-]\d{2}:\d{2}$/, '')
          const [datePart, timePart] = stripped.split('T')
          setEditDate(datePart ?? '')
          if (timePart) {
            const [h, m] = timePart.split(':')
            setEditTime(`${h}:${m}`)
          }
        }

        // og meta tags for JS-capable crawlers (Twitter, Slack, Discord)
        const setMeta = (property: string, content: string) => {
          let el = document.querySelector(`meta[property="${property}"]`) as HTMLMetaElement | null
          if (!el) {
            el = document.createElement('meta')
            el.setAttribute('property', property)
            document.head.appendChild(el)
          }
          el.setAttribute('content', content)
        }
        const origin = window.location.origin
        const s = data.event.slug
        setMeta('og:title', data.event.title)
        setMeta('og:description', "Tap to see who's coming →")
        setMeta('og:image', `https://ollae.app/og/${s}`)
        setMeta('og:image:width', '1200')
        setMeta('og:image:height', '630')
        setMeta('og:url', `${origin}/events/${s}`)
        setMeta('og:type', 'website')
      })
      .catch(e => {
        setError(e.message)
        setLoading(false)
      })
  }, [slug])

  async function handleSaveEdit() {
    if (!slug || !adminToken || !editTitle.trim()) return
    setSaving(true)
    try {
      const eventDate = editDate
        ? (editTime ? `${editDate}T${editTime}:00` : `${editDate}T00:00:00`)
        : undefined
      const res = await fetch(`${API}/events/${slug}?admin=${adminToken}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: editTitle.trim(),
          location: editLocation.trim() || undefined,
          event_date: eventDate,
        }),
      })
      if (!res.ok) throw new Error('Failed to save')
      const updated: Event = await res.json()
      setEvent(updated)
      setIsEditing(false)
    } catch {
      alert('Something went wrong. Try again.')
    } finally {
      setSaving(false)
    }
  }

  async function handleSubmit() {
    if (!name.trim() || !status) return
    setSubmitting(true)
    try {
      const res = await fetch(`${API}/events/${slug}/rsvp`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name.trim(), status, guests }),
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

  if (submitted) return <SuccessScreen name={name} status={status!} guests={guests} onBack={() => setSubmitted(false)} />

  const attendingCount = responses.filter(r => r.status === 'in').reduce((sum, r) => sum + 1 + (r.guests || 0), 0)
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
            {event!.title}
          </h1>
          {(() => {
            const parts: string[] = []
            if (event!.event_date) parts.push(`📅 ${formatDate(event!.event_date)}`)
            if (event!.event_date && hasTime(event!.event_date)) parts.push(`⏰ ${formatTime(event!.event_date)}`)
            if (event!.location) parts.push(`📍 ${event!.location}`)
            if (parts.length === 0) return null
            return <p className="text-[13px] font-bold text-text-muted mt-1">{parts.join(' · ')}</p>
          })()}
          {isAdmin && !isEditing && (
            <button
              onClick={() => setIsEditing(true)}
              className="self-start text-xs text-text-muted underline underline-offset-2 hover:opacity-70 transition-opacity mt-1"
            >
              ✏️ Edit event
            </button>
          )}
        </div>

        {/* Admin edit form */}
        {isAdmin && isEditing && (
          <div className="flex flex-col gap-3 bg-bg-elevated rounded-2xl p-5">
            <p className="text-xs font-semibold text-text-muted uppercase tracking-wide">Edit Event</p>
            <div className="flex flex-col gap-2">
              <label className="text-xs text-text-primary">Event Name</label>
              <input
                type="text"
                value={editTitle}
                onChange={e => setEditTitle(e.target.value)}
                className="w-full bg-bg-surface rounded-xl p-4 text-base text-text-primary border border-white/[0.08] focus:outline-none focus:border-white/25 transition-colors [color-scheme:dark]"
              />
            </div>
            <div className="flex gap-3">
              <div className="flex flex-col gap-2 flex-1 min-w-0">
                <label className="text-xs text-text-primary">Date</label>
                <input
                  type="date"
                  value={editDate}
                  onChange={e => setEditDate(e.target.value)}
                  className="w-full bg-bg-surface rounded-xl p-4 text-base text-text-primary border border-white/[0.08] focus:outline-none focus:border-white/25 transition-colors [color-scheme:dark]"
                />
              </div>
              <div className="flex flex-col gap-2 flex-1 min-w-0">
                <label className="text-xs text-text-primary">Time</label>
                <input
                  type="time"
                  value={editTime}
                  onChange={e => setEditTime(e.target.value)}
                  className="w-full bg-bg-surface rounded-xl p-4 text-base text-text-primary border border-white/[0.08] focus:outline-none focus:border-white/25 transition-colors [color-scheme:dark]"
                />
              </div>
            </div>
            <div className="flex flex-col gap-2">
              <label className="text-xs text-text-primary">Location</label>
              <input
                type="text"
                value={editLocation}
                onChange={e => setEditLocation(e.target.value)}
                className="w-full bg-bg-surface rounded-xl p-4 text-base text-text-primary border border-white/[0.08] focus:outline-none focus:border-white/25 transition-colors [color-scheme:dark]"
              />
            </div>
            <div className="flex gap-3 mt-1">
              <button
                onClick={handleSaveEdit}
                disabled={!editTitle.trim() || saving}
                className="flex-1 py-3 rounded-xl text-base font-normal border transition-colors bg-[#F59E0B] text-[#F8FAFC] border-[#F8FAFC] hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? 'Saving…' : 'Save Changes'}
              </button>
              <button
                onClick={() => setIsEditing(false)}
                className="flex-1 py-3 rounded-xl text-base font-normal border border-white/[0.08] text-text-muted hover:opacity-70 transition-opacity"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

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
                      {' '}{statusText(r.status, r.guests)}{' · '}{timeAgo(r.created_at)}
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
                onClick={() => handleSetStatus('in')}
                className={`flex-1 flex flex-col items-center justify-center gap-4 py-5 px-8 rounded-xl border transition-colors ${
                  status === 'in'
                    ? 'bg-[#DCFCE7] border-[#16A34A]'
                    : 'bg-bg-elevated border-white/[0.08]'
                }`}
              >
                <span className="text-[32px] leading-none">✅</span>
                <span className={`text-base font-normal ${status === 'in' ? 'text-[#22C55E]' : 'text-text-primary'}`}>
                  I'm in
                </span>
              </button>

              {/* Can't make it */}
              <button
                onClick={() => handleSetStatus('out')}
                className={`flex-1 flex flex-col items-center justify-center gap-4 py-5 px-8 rounded-xl border transition-colors ${
                  status === 'out'
                    ? 'bg-[#FEE2E2] border-[#DC2626]'
                    : 'bg-bg-elevated border-white/[0.08]'
                }`}
              >
                <span className={`text-[32px] leading-none ${status === 'out' ? '' : 'opacity-60'}`}>😔</span>
                <span className={`text-base font-normal ${status === 'out' ? 'text-[#EF4444]' : 'text-text-primary'}`}>
                  Can't make it
                </span>
              </button>
            </div>

            {/* Guest picker — shown when I'm in */}
            {status === 'in' && (
              <div className="flex flex-col gap-3">
                <p className="text-sm text-text-muted text-center">How many spots do you need?</p>
                <div className="flex gap-2 justify-center flex-wrap">
                  {[
                    { label: 'Just me', value: 0 },
                    { label: '+1', value: 1 },
                    { label: '+2', value: 2 },
                    { label: '+3', value: 3 },
                    { label: '+4', value: 4 },
                    { label: '+5', value: 5 },
                  ].map(opt => (
                    <button
                      key={opt.value}
                      onClick={() => { setGuests(opt.value); setShowCustomGuests(false) }}
                      className={`px-4 py-2 rounded-full text-sm font-medium border transition-colors ${
                        guests === opt.value && !showCustomGuests
                          ? 'bg-[#22C55E] border-[#16A34A] text-white'
                          : 'bg-bg-elevated border-white/[0.08] text-text-primary hover:border-white/25'
                      }`}
                    >
                      {opt.label}
                    </button>
                  ))}
                  <button
                    onClick={() => { setShowCustomGuests(true); if (guests <= 5) setGuests(6) }}
                    className={`px-4 py-2 rounded-full text-sm font-medium border transition-colors ${
                      showCustomGuests
                        ? 'bg-[#22C55E] border-[#16A34A] text-white'
                        : 'bg-bg-elevated border-white/[0.08] text-text-primary hover:border-white/25'
                    }`}
                  >
                    More
                  </button>
                </div>
                {showCustomGuests && (
                  <div className="flex items-center gap-2 justify-center">
                    <span className="text-sm text-text-muted">+</span>
                    <input
                      type="number"
                      min="1"
                      max="99"
                      value={guests}
                      onChange={e => {
                        const n = parseInt(e.target.value, 10)
                        if (!isNaN(n) && n >= 1) setGuests(n)
                      }}
                      autoFocus
                      className="w-20 bg-bg-surface rounded-xl px-3 py-2 text-base text-text-primary border border-[#22C55E] focus:outline-none text-center [color-scheme:dark]"
                    />
                    <span className="text-sm text-text-muted">extra guests</span>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Not sure yet */}
          <div className="flex justify-center">
            <button
              onClick={() => handleSetStatus('remind_me')}
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
            {submitting ? 'Submitting...' : status === 'in' && guests > 0 ? "We're in! →" : 'Submit →'}
          </button>
        </div>

      </div>
    </div>
  )
}

function SuccessScreen({ name, status, guests, onBack }: { name: string; status: RSVPStatus; guests: number; onBack: () => void }) {
  const emoji = status === 'in' ? '🙌' : status === 'out' ? '😢' : '🔔'
  const message = status === 'in' ? "You're on the list!" : status === 'out' ? "Got it, you're out." : "We'll remind you!"
  const guestSuffix = status === 'in' && guests > 0 ? ` (+${guests})` : ''
  const statusLine = status === 'in' ? `${name}${guestSuffix} · I'm in 🏐` : status === 'out' ? `${name} · Can't make it 😔` : `${name} · Remind me 🔔`
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

function statusText(status: string, guests?: number) {
  if (status === 'in') return guests && guests > 0 ? `is in (+${guests}) 🏐` : 'is in 🏐'
  if (status === 'out') return "can't make it 😔"
  if (status === 'remind_me') return 'wants a reminder 🔔'
  return status
}

// Strip any timezone suffix (Z or ±HH:MM) so JS treats the timestamp as
// the wall-clock time the organizer entered, not a UTC moment to convert.
// Postgres stores TIMESTAMPTZ and returns a Z-suffixed string, but for
// ollae events the stored time is already the intended local time.
function parseAsWallClock(dateStr: string) {
  return new Date(dateStr.replace(/Z$|[+-]\d{2}:\d{2}$/, ''))
}

function formatDate(dateStr: string) {
  return parseAsWallClock(dateStr).toLocaleDateString('en-US', {
    weekday: 'long', month: 'long', day: 'numeric',
  })
}

function formatTime(dateStr: string) {
  return parseAsWallClock(dateStr).toLocaleTimeString('en-US', {
    hour: 'numeric', minute: '2-digit', hour12: true,
  })
}

function hasTime(dateStr: string) {
  const d = parseAsWallClock(dateStr)
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
