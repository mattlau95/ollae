import { ImageResponse } from '@vercel/og'

const BACKEND = 'https://ollae-backend.fly.dev'

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', {
    weekday: 'short', month: 'short', day: 'numeric',
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

export default async function handler(req: Request) {
  const slug = new URL(req.url).pathname.split('/').filter(Boolean).pop()
  if (!slug) return new Response('Not found', { status: 404 })

  const [interBold, interMedium, eventData] = await Promise.all([
    fetch('https://fonts.bunny.net/inter/files/inter-latin-700-normal.ttf').then(r => r.arrayBuffer()),
    fetch('https://fonts.bunny.net/inter/files/inter-latin-500-normal.ttf').then(r => r.arrayBuffer()),
    fetch(`${BACKEND}/events/${slug}`).then(r => r.ok ? r.json() : null).catch(() => null),
  ])

  if (!eventData?.event) return new Response('Not found', { status: 404 })

  const { event } = eventData
  const dateStr = event.event_date ? formatDate(event.event_date) : null
  const timeStr = event.event_date && hasTime(event.event_date) ? formatTime(event.event_date) : null
  const whenStr = [dateStr, timeStr].filter(Boolean).join(' · ')
  const fontSize = event.title.length > 28 ? 54 : event.title.length > 20 ? 64 : 72

  return new ImageResponse(
    (
      <div
        style={{
          width: 1200,
          height: 630,
          backgroundColor: '#0F172A',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: '48px',
          fontFamily: 'Inter',
        }}
      >
        {/* Top: pill badge */}
        <div style={{ display: 'flex' }}>
          <div
            style={{
              backgroundColor: 'rgba(245, 158, 11, 0.15)',
              border: '1.5px solid #F59E0B',
              borderRadius: '999px',
              padding: '8px 16px',
              color: '#F59E0B',
              fontSize: 14,
              fontWeight: 700,
              letterSpacing: '0.08em',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            {event.title.toUpperCase()}
          </div>
        </div>

        {/* Center: title + tagline */}
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px' }}>
          <div
            style={{
              fontSize,
              fontWeight: 700,
              color: '#F8FAFC',
              textAlign: 'center',
              lineHeight: '1',
            }}
          >
            {event.title}
          </div>
          <div style={{ fontSize: 22, fontWeight: 500, color: '#64748B' }}>
            Tap to see who's coming →
          </div>
        </div>

        {/* Bottom: meta + button */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
          <div style={{ display: 'flex', gap: '40px' }}>
            {whenStr && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                <div style={{ fontSize: 14, fontWeight: 500, color: '#64748B', letterSpacing: '0.1em' }}>
                  WHEN
                </div>
                <div style={{ fontSize: 20, fontWeight: 500, color: '#94A3B8' }}>
                  {whenStr}
                </div>
              </div>
            )}
            {event.location && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                <div style={{ fontSize: 14, fontWeight: 500, color: '#64748B', letterSpacing: '0.1em' }}>
                  WHERE
                </div>
                <div style={{ fontSize: 20, fontWeight: 500, color: '#94A3B8' }}>
                  {event.location}
                </div>
              </div>
            )}
          </div>

          <div
            style={{
              backgroundColor: '#F59E0B',
              borderRadius: '12px',
              padding: '16px 24px',
              fontSize: 16,
              fontWeight: 700,
              color: '#000000',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            RSVP Now →
          </div>
        </div>
      </div>
    ),
    {
      width: 1200,
      height: 630,
      fonts: [
        { name: 'Inter', data: interBold, weight: 700, style: 'normal' },
        { name: 'Inter', data: interMedium, weight: 500, style: 'normal' },
      ],
    }
  )
}
