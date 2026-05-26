import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

const API = window.location.hostname === 'localhost' ? 'http://localhost:8080' : 'https://ollae-backend.fly.dev'

export default function CreatePage() {
  useEffect(() => { document.title = 'Create Event · ollae.app' }, [])

  const [view, setView] = useState<'nl' | 'form'>('nl')
  const [nlInput, setNlInput] = useState('')
  const [parsing, setParsing] = useState(false)
  const [assumed, setAssumed] = useState<Set<string>>(new Set())

  const [title, setTitle] = useState('')
  const [location, setLocation] = useState('')
  const [date, setDate] = useState('')
  const [time, setTime] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [slug, setSlug] = useState<string | null>(null)
  const [isEditing, setIsEditing] = useState(false)
  const [copied, setCopied] = useState(false)

  const eventUrl = slug ? `${window.location.origin}/events/${slug}` : ''
  const displayUrl = slug ? `${window.location.hostname}/events/${slug}` : ''
  const canCreate = title.trim().length > 0

  function buildEventDate() {
    if (!date) return undefined
    const raw = time ? `${date}T${time}` : `${date}T00:00`
    return new Date(raw).toISOString()
  }

  async function handleParse() {
    if (!nlInput.trim()) return
    setParsing(true)
    try {
      const res = await fetch(`${API}/parse-event`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: nlInput.trim() }),
      })
      if (!res.ok) throw new Error()
      const parsed = await res.json()

      const newAssumed = new Set<string>()
      setTitle(parsed.title ?? '')
      if (parsed.title) newAssumed.add('title')
      setDate(parsed.date ?? '')
      if (parsed.date) newAssumed.add('date')
      setTime(parsed.time ?? '')
      if (parsed.time) newAssumed.add('time')
      setLocation(parsed.location ?? '')
      if (parsed.location) newAssumed.add('location')
      setAssumed(newAssumed)
      setView('form')
    } catch {
      alert('Could not parse. Try again or fill in manually.')
    } finally {
      setParsing(false)
    }
  }

  function goManual() {
    setTitle('')
    setDate('')
    setTime('')
    setLocation('')
    setAssumed(new Set())
    setView('form')
  }

  function clearAssumed(field: string) {
    setAssumed(prev => { const s = new Set(prev); s.delete(field); return s })
  }

  async function handleCreate() {
    if (!canCreate) return
    setSubmitting(true)
    try {
      const res = await fetch(`${API}/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: title.trim(),
          location: location.trim() || undefined,
          event_date: buildEventDate(),
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

  async function handleUpdate() {
    if (!slug || !canCreate) return
    setSubmitting(true)
    try {
      const res = await fetch(`${API}/events/${slug}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: title.trim(),
          location: location.trim() || undefined,
          event_date: buildEventDate(),
        }),
      })
      if (!res.ok) throw new Error('Failed to update event')
      setIsEditing(false)
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

  const isLocked = !!slug && !isEditing
  const baseInput = `w-full bg-bg-surface rounded-xl p-4 text-base text-text-primary placeholder-text-muted border focus:outline-none transition-colors [color-scheme:dark]`
  function fieldClass(field: string) {
    return assumed.has(field)
      ? `${baseInput} border-[#F59E0B]/70 focus:border-[#F59E0B]`
      : `${baseInput} border-white/[0.08] focus:border-white/25`
  }

  return (
    <div className="min-h-screen bg-bg-base flex flex-col items-center px-4 py-6 sm:py-12 overflow-y-auto">
      <div className="w-full max-w-sm flex flex-col gap-5 flex-1">

        <Link to="/create">
          <img src="/ollae-logo.svg" alt="ollae" className="h-7 w-auto" />
        </Link>

        {/* NL view */}
        {view === 'nl' && !slug && (
          <div className="flex flex-col gap-4 flex-1">
            <div className="flex flex-col gap-2">
              <h1 className="text-[28px] font-semibold text-text-primary leading-tight">Create Event</h1>
              <p className="text-sm text-text-muted">Describe it — our integrated Claude API will fill in the details.</p>
            </div>
            <textarea
              placeholder={"e.g. Volleyball this Saturday at 2pm, Venice Beach Court 4"}
              value={nlInput}
              onChange={e => setNlInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleParse() } }}
              className={`${baseInput} resize-none h-28`}
            />
            <button
              onClick={handleParse}
              disabled={!nlInput.trim() || parsing}
              className={`w-full py-4 rounded-xl text-xl font-normal border transition-colors ${
                nlInput.trim()
                  ? 'bg-[#F59E0B] text-[#F8FAFC] border-[#F8FAFC] hover:opacity-90'
                  : 'bg-bg-surface text-text-muted border-white/[0.08] cursor-not-allowed'
              }`}
            >
              {parsing ? 'Creating...' : 'Create Event →'}
            </button>
            <div className="flex justify-center">
              <button
                onClick={goManual}
                className="text-sm text-text-muted underline underline-offset-2 hover:opacity-70 transition-opacity"
              >
                Fill in manually instead
              </button>
            </div>
          </div>
        )}

        {/* Form view */}
        {(view === 'form' || !!slug) && (
          <>
            {!slug && nlInput && (
              <button
                onClick={() => setView('nl')}
                className="text-sm text-text-muted underline underline-offset-2 hover:opacity-70 transition-opacity self-start"
              >
                ← Try a different description
              </button>
            )}

            <div className={`flex flex-col gap-2 transition-opacity ${isLocked ? 'opacity-[0.45]' : ''}`}>
              <h1 className="text-[28px] font-semibold text-text-primary leading-tight">Create Event</h1>
              <p className="text-sm text-text-muted">Fill in the details below, get a shareable link.</p>
            </div>

            <div className={`flex flex-col gap-3 transition-opacity ${isLocked ? 'opacity-[0.45] pointer-events-none' : ''}`}>
              <div className="flex flex-col gap-2">
                <label className="text-xs font-normal text-text-primary">Event Name</label>
                <input
                  type="text"
                  placeholder="Pickup Volleyball"
                  value={title}
                  onChange={e => { setTitle(e.target.value); clearAssumed('title') }}
                  className={fieldClass('title')}
                />
              </div>

              <div className="flex flex-col gap-2">
                <label className="text-xs font-normal text-text-primary">Date</label>
                <input
                  type="date"
                  value={date}
                  onChange={e => { setDate(e.target.value); clearAssumed('date') }}
                  className={fieldClass('date')}
                />
              </div>

              <div className="flex flex-col gap-2">
                <label className="text-xs font-normal text-text-primary">Time</label>
                <input
                  type="time"
                  value={time}
                  onChange={e => { setTime(e.target.value); clearAssumed('time') }}
                  className={fieldClass('time')}
                />
              </div>

              <div className="flex flex-col gap-2">
                <label className="text-xs font-normal text-text-primary">Location</label>
                <input
                  type="text"
                  placeholder="Venice Beach Court 4"
                  value={location}
                  onChange={e => { setLocation(e.target.value); clearAssumed('location') }}
                  className={fieldClass('location')}
                />
              </div>
            </div>

            <button
              onClick={
                isEditing ? handleUpdate :
                slug ? () => setIsEditing(true) :
                handleCreate
              }
              disabled={(!slug || isEditing) && (!canCreate || submitting)}
              className={`w-full py-4 rounded-xl text-xl font-normal border transition-colors ${
                slug || canCreate
                  ? 'bg-[#F59E0B] text-[#F8FAFC] border-[#F8FAFC] hover:opacity-90'
                  : 'bg-bg-surface text-text-muted border-white/[0.08] cursor-not-allowed'
              }`}
            >
              {isEditing
                ? (submitting ? 'Saving...' : 'Save Changes ->')
                : slug
                ? 'Edit Event Details'
                : (submitting ? 'Creating...' : 'Create Event ->')}
            </button>

            <div className="flex-1 max-h-12" />

            {slug && (
              <div className="flex flex-col gap-3">
                <h2 className="text-[28px] font-semibold text-text-primary leading-tight">Share this link</h2>
                <button
                  onClick={handleCopy}
                  className="bg-bg-elevated rounded-2xl p-6 flex flex-col gap-2 text-left w-full active:opacity-70 transition-opacity"
                >
                  <p className="text-[11px] font-semibold text-text-muted text-center w-full uppercase tracking-wide">
                    Your shareable link
                  </p>
                  <p className="text-base font-semibold text-[#F59E0B] text-center w-full truncate">
                    {displayUrl}
                  </p>
                  <div className="flex items-center justify-center gap-2 mt-1">
                    <CopyIcon />
                    <span className="text-base font-semibold text-text-muted">
                      {copied ? 'Copied!' : 'tap to copy'}
                    </span>
                  </div>
                </button>
              </div>
            )}
          </>
        )}

      </div>
    </div>
  )
}

function CopyIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="7" y="7" width="10" height="10" rx="2" stroke="#94A3B8" strokeWidth="1.5" />
      <path d="M13 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2" stroke="#94A3B8" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}
