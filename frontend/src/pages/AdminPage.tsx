import { useEffect, useState } from 'react'

const API = window.location.hostname === 'localhost' ? 'http://localhost:8080' : 'https://ollae-backend.fly.dev'
const STORAGE_KEY = 'ollae_admin_key'

type AdminResponse = {
  id: string
  event_id: string
  name: string
  status: 'in' | 'out' | 'remind_me'
  guests: number
  created_at: string
}

type AdminEvent = {
  id: string
  slug: string
  title: string
  location: string
  event_date: string | null
  created_at: string
  emoji: string
  counts: { in: number; out: number; remind_me: number }
  responses: AdminResponse[]
}

type AdminData = {
  stats: { total_events: number; total_rsvps: number; active_events: number }
  retention_months: number
  events: AdminEvent[]
}

function authHeaders(key: string) {
  return { Authorization: `Bearer ${key}` }
}

function formatDate(dateStr: string | null) {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

export default function AdminPage() {
  const [key, setKey] = useState<string | null>(() => sessionStorage.getItem(STORAGE_KEY))
  const [keyInput, setKeyInput] = useState('')
  const [authError, setAuthError] = useState(false)
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<AdminData | null>(null)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [editingSlug, setEditingSlug] = useState<string | null>(null)
  const [editForm, setEditForm] = useState({ title: '', location: '', event_date: '' })
  const [editSaving, setEditSaving] = useState(false)
  const [retentionInput, setRetentionInput] = useState('')
  const [retentionSaving, setRetentionSaving] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [batchDeleting, setBatchDeleting] = useState(false)
  const [rescraping, setRescraping] = useState(false)
  const [rescrapeStatus, setRescrapeStatus] = useState<string | null>(null)

  useEffect(() => {
    if (key) fetchData(key)
  }, [key])

  async function fetchData(k: string) {
    setLoading(true)
    try {
      const res = await fetch(`${API}/admin/events`, { headers: authHeaders(k) })
      if (res.status === 401) {
        sessionStorage.removeItem(STORAGE_KEY)
        setKey(null)
        setAuthError(true)
        return
      }
      if (!res.ok) throw new Error()
      const json: AdminData = await res.json()
      setData(json)
      setRetentionInput(String(json.retention_months))
    } catch {
      alert('Failed to load data.')
    } finally {
      setLoading(false)
    }
  }

  function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    const k = keyInput.trim()
    if (!k) return
    sessionStorage.setItem(STORAGE_KEY, k)
    setKey(k)
    setKeyInput('')
    setAuthError(false)
  }

  function handleLogout() {
    sessionStorage.removeItem(STORAGE_KEY)
    setKey(null)
    setData(null)
  }

  async function deleteEvent(slug: string, title: string) {
    if (!window.confirm(`Delete "${title}" and all its responses?`)) return
    await fetch(`${API}/admin/events/${slug}`, { method: 'DELETE', headers: authHeaders(key!) })
    setData(prev => prev ? { ...prev, events: prev.events.filter(e => e.slug !== slug) } : null)
  }

  async function deleteResponse(eventSlug: string, responseId: string, name: string) {
    if (!window.confirm(`Delete ${name}'s response?`)) return
    await fetch(`${API}/admin/responses/${responseId}`, { method: 'DELETE', headers: authHeaders(key!) })
    setData(prev => {
      if (!prev) return null
      return {
        ...prev,
        events: prev.events.map(e =>
          e.slug === eventSlug
            ? { ...e, responses: e.responses.filter(r => r.id !== responseId) }
            : e
        ),
      }
    })
  }

  function startEdit(ev: AdminEvent) {
    setEditingSlug(ev.slug)
    setEditForm({
      title: ev.title,
      location: ev.location || '',
      event_date: ev.event_date ? ev.event_date.slice(0, 16) : '',
    })
  }

  async function saveEdit(slug: string) {
    setEditSaving(true)
    try {
      const res = await fetch(`${API}/admin/events/${slug}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', ...authHeaders(key!) },
        body: JSON.stringify({
          title: editForm.title,
          location: editForm.location || undefined,
          event_date: editForm.event_date || undefined,
        }),
      })
      if (!res.ok) throw new Error()
      const updated = await res.json()
      setData(prev => prev ? {
        ...prev,
        events: prev.events.map(e => e.slug === slug ? { ...e, ...updated } : e),
      } : null)
      setEditingSlug(null)
    } catch {
      alert('Failed to save.')
    } finally {
      setEditSaving(false)
    }
  }

  async function saveRetention() {
    const months = parseInt(retentionInput)
    if (isNaN(months) || months < 1 || months > 24) {
      alert('Retention must be 1–24 months.')
      return
    }
    setRetentionSaving(true)
    try {
      await fetch(`${API}/admin/settings`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', ...authHeaders(key!) },
        body: JSON.stringify({ retention_months: months }),
      })
      setData(prev => prev ? { ...prev, retention_months: months } : null)
    } catch {
      alert('Failed to save retention setting.')
    } finally {
      setRetentionSaving(false)
    }
  }

  function toggleExpand(slug: string) {
    setExpanded(prev => {
      const next = new Set(prev)
      next.has(slug) ? next.delete(slug) : next.add(slug)
      return next
    })
  }

  function toggleSelect(slug: string) {
    setSelected(prev => {
      const next = new Set(prev)
      next.has(slug) ? next.delete(slug) : next.add(slug)
      return next
    })
  }

  function selectAll() {
    setSelected(new Set(data?.events.map(e => e.slug) ?? []))
  }

  function deselectAll() {
    setSelected(new Set())
  }

  async function rescrapeSelected() {
    if (selected.size === 0) return
    setRescraping(true)
    setRescrapeStatus(null)
    const slugs = [...selected]
    let done = 0
    await Promise.all(slugs.map(async slug => {
      await fetch(`${API}/events/${slug}/rescrape`, { method: 'POST' })
      done++
      setRescrapeStatus(`${done}/${slugs.length}`)
    }))
    setRescrapeStatus('Done ✓')
    setRescraping(false)
    setTimeout(() => setRescrapeStatus(null), 4000)
  }

  async function deleteSelected() {
    if (selected.size === 0) return
    if (!window.confirm(`Delete ${selected.size} event${selected.size !== 1 ? 's' : ''} and all their responses?`)) return
    setBatchDeleting(true)
    const slugs = [...selected]
    await Promise.all(slugs.map(slug =>
      fetch(`${API}/admin/events/${slug}`, { method: 'DELETE', headers: authHeaders(key!) })
    ))
    setData(prev => prev ? { ...prev, events: prev.events.filter(e => !selected.has(e.slug)) } : null)
    setSelected(new Set())
    setBatchDeleting(false)
  }

  if (!key) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center px-4">
        <form onSubmit={handleLogin} className="flex flex-col gap-4 w-full max-w-xs">
          <h1 className="text-xl font-semibold text-gray-100">ollae admin</h1>
          {authError && <p className="text-sm text-red-400">Incorrect password.</p>}
          <input
            type="password"
            placeholder="Password"
            value={keyInput}
            onChange={e => setKeyInput(e.target.value)}
            autoFocus
            className="bg-gray-800 text-gray-100 rounded-lg px-4 py-3 text-sm border border-gray-700 focus:outline-none focus:border-gray-500 [color-scheme:dark]"
          />
          <button
            type="submit"
            className="bg-amber-500 text-white rounded-lg px-4 py-3 text-sm font-medium hover:opacity-90 transition-opacity"
          >
            Sign in
          </button>
        </form>
      </div>
    )
  }

  if (loading || !data) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <p className="text-gray-500 text-sm">Loading...</p>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100 p-4 sm:p-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
        <div className="flex items-center gap-4 flex-wrap">
          <h1 className="text-lg font-semibold">ollae admin</h1>
          <span className="text-sm text-gray-400">
            {data.stats.total_events} events · {data.stats.total_rsvps} RSVPs · {data.stats.active_events} active
          </span>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm text-gray-400">Retention:</span>
          <input
            type="number"
            min={1}
            max={24}
            value={retentionInput}
            onChange={e => setRetentionInput(e.target.value)}
            className="w-14 bg-gray-800 text-gray-100 rounded px-2 py-1 text-sm border border-gray-700 focus:outline-none focus:border-gray-500 [color-scheme:dark]"
          />
          <span className="text-sm text-gray-400">months</span>
          <button
            onClick={saveRetention}
            disabled={retentionSaving}
            className="text-sm text-amber-400 hover:text-amber-300 transition-colors disabled:opacity-50"
          >
            {retentionSaving ? 'Saving…' : 'Save'}
          </button>
          <span className="text-gray-700 mx-1">|</span>
          <button onClick={handleLogout} className="text-sm text-gray-500 hover:text-gray-300 transition-colors">
            Log out
          </button>
        </div>
      </div>

      {/* Events */}
      <div className="flex flex-col gap-2">
        {/* Batch action bar */}
        {data.events.length > 0 && (
          <div className="flex items-center gap-3 px-1 pb-1 text-sm">
            <input
              type="checkbox"
              checked={selected.size === data.events.length && data.events.length > 0}
              ref={el => { if (el) el.indeterminate = selected.size > 0 && selected.size < data.events.length }}
              onChange={e => e.target.checked ? selectAll() : deselectAll()}
              className="accent-amber-500 cursor-pointer"
              title="Select all"
            />
            {selected.size > 0 ? (
              <>
                <span className="text-gray-400">{selected.size} selected</span>
                <button
                  onClick={rescrapeSelected}
                  disabled={rescraping || batchDeleting}
                  className="text-amber-400 hover:text-amber-300 disabled:opacity-50 transition-colors font-medium"
                >
                  {rescraping ? `Rescraping… ${rescrapeStatus ?? ''}` : `Rescrape FB (${selected.size})`}
                </button>
                {!rescraping && rescrapeStatus && (
                  <span className="text-green-400 text-xs">{rescrapeStatus}</span>
                )}
                <button
                  onClick={deleteSelected}
                  disabled={batchDeleting || rescraping}
                  className="text-red-400 hover:text-red-300 disabled:opacity-50 transition-colors font-medium"
                >
                  {batchDeleting ? 'Deleting…' : `Delete selected (${selected.size})`}
                </button>
                <button onClick={deselectAll} className="text-gray-600 hover:text-gray-400 transition-colors">
                  Deselect all
                </button>
              </>
            ) : (
              <span className="text-gray-600">Select all</span>
            )}
          </div>
        )}

        {data.events.length === 0 && (
          <p className="text-gray-500 text-sm">No events yet.</p>
        )}
        {data.events.map(ev => (
          <div key={ev.slug} className="bg-gray-800 rounded-lg overflow-hidden">
            {/* Event row */}
            <div className="flex items-center gap-3 px-4 py-3">
              <input
                type="checkbox"
                checked={selected.has(ev.slug)}
                onChange={() => toggleSelect(ev.slug)}
                className="accent-amber-500 cursor-pointer shrink-0"
              />
              <span className="text-xl shrink-0">{ev.emoji || '📅'}</span>

              <div className="flex-1 min-w-0">
                {editingSlug === ev.slug ? (
                  <div className="flex flex-col gap-2">
                    <input
                      value={editForm.title}
                      onChange={e => setEditForm(f => ({ ...f, title: e.target.value }))}
                      className="bg-gray-700 text-gray-100 rounded px-2 py-1 text-sm w-full border border-gray-600 focus:outline-none focus:border-gray-400"
                    />
                    <div className="flex gap-2 flex-wrap">
                      <input
                        value={editForm.location}
                        onChange={e => setEditForm(f => ({ ...f, location: e.target.value }))}
                        placeholder="Location"
                        className="bg-gray-700 text-gray-100 rounded px-2 py-1 text-sm flex-1 border border-gray-600 focus:outline-none focus:border-gray-400"
                      />
                      <input
                        type="datetime-local"
                        value={editForm.event_date}
                        onChange={e => setEditForm(f => ({ ...f, event_date: e.target.value }))}
                        className="bg-gray-700 text-gray-100 rounded px-2 py-1 text-sm border border-gray-600 focus:outline-none focus:border-gray-400 [color-scheme:dark]"
                      />
                    </div>
                    <div className="flex gap-3">
                      <button
                        onClick={() => saveEdit(ev.slug)}
                        disabled={editSaving}
                        className="text-xs text-amber-400 hover:text-amber-300 disabled:opacity-50 transition-colors"
                      >
                        {editSaving ? 'Saving…' : 'Save'}
                      </button>
                      <button
                        onClick={() => setEditingSlug(null)}
                        className="text-xs text-gray-500 hover:text-gray-300 transition-colors"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-baseline gap-2 flex-wrap">
                    <button
                      onClick={() => startEdit(ev)}
                      className="text-sm font-medium text-gray-100 hover:text-amber-400 transition-colors text-left"
                    >
                      {ev.title}
                    </button>
                    {ev.location && <span className="text-xs text-gray-500">{ev.location}</span>}
                    <span className="text-xs text-gray-500">{formatDate(ev.event_date)}</span>
                    <span className="text-xs text-gray-600">{ev.slug}</span>
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2 shrink-0 text-xs">
                <span className="text-green-400">{ev.counts.in}in</span>
                <span className="text-red-400">{ev.counts.out}out</span>
                <span className="text-amber-400">{ev.counts.remind_me}?</span>
                <button
                  onClick={() => toggleExpand(ev.slug)}
                  className="text-gray-500 hover:text-gray-300 transition-colors ml-1 text-sm w-4"
                  title={expanded.has(ev.slug) ? 'Collapse' : 'Expand'}
                >
                  {expanded.has(ev.slug) ? '↑' : '↓'}
                </button>
                <button
                  onClick={() => deleteEvent(ev.slug, ev.title)}
                  className="text-gray-600 hover:text-red-400 transition-colors text-sm"
                  title="Delete event"
                >
                  ✕
                </button>
              </div>
            </div>

            {/* Expanded responses */}
            {expanded.has(ev.slug) && (
              <div className="border-t border-gray-700">
                {ev.responses.length === 0 ? (
                  <p className="text-xs text-gray-500 px-4 py-3">No responses yet.</p>
                ) : (
                  ev.responses.map(resp => (
                    <div key={resp.id} className="flex items-center gap-3 px-4 py-2 border-b border-gray-700/40 last:border-0">
                      <span className="text-sm text-gray-300 flex-1 min-w-0 truncate">
                        <span className="font-medium">{resp.name}</span>
                        {resp.guests > 0 && <span className="text-gray-500"> +{resp.guests}</span>}
                        {' · '}
                        <span className={
                          resp.status === 'in' ? 'text-green-400' :
                          resp.status === 'out' ? 'text-red-400' : 'text-amber-400'
                        }>
                          {resp.status === 'in' ? 'in' : resp.status === 'out' ? 'out' : 'remind'}
                        </span>
                        {' · '}
                        <span className="text-gray-500 text-xs">{timeAgo(resp.created_at)}</span>
                      </span>
                      <button
                        onClick={() => deleteResponse(ev.slug, resp.id, resp.name)}
                        className="text-gray-600 hover:text-red-400 transition-colors text-xs shrink-0"
                        title="Delete response"
                      >
                        ✕
                      </button>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
