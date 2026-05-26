const BACKEND = 'https://ollae-backend.fly.dev'

const CRAWLERS =
  /WhatsApp|Telegram|Twitterbot|Slackbot|Discordbot|facebookexternalhit|LinkedInBot|Googlebot|Applebot|iMessage|TelegramBot/i

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

export default async function middleware(request: Request): Promise<Response | undefined> {
  const url = new URL(request.url)
  const match = url.pathname.match(/^\/events\/([^/]+)$/)
  if (!match) return undefined

  const ua = request.headers.get('user-agent') || ''
  if (!CRAWLERS.test(ua)) return undefined

  const slug = match[1]

  const eventData = await fetch(`${BACKEND}/events/${slug}`)
    .then(r => (r.ok ? r.json() : null))
    .catch(() => null)

  if (!eventData?.event) return undefined

  const { event } = eventData
  const title = escapeHtml(event.title)
  const ogImage = `${url.origin}/api/og/${slug}`
  const ogUrl = `${url.origin}/events/${slug}`
  const description = "Tap to see who's coming →"

  const html = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>${title} · ollae.app</title>
  <meta property="og:title" content="${title}" />
  <meta property="og:description" content="${escapeHtml(description)}" />
  <meta property="og:image" content="${ogImage}" />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
  <meta property="og:url" content="${ogUrl}" />
  <meta property="og:type" content="website" />
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content="${title}" />
  <meta name="twitter:description" content="${escapeHtml(description)}" />
  <meta name="twitter:image" content="${ogImage}" />
  <meta http-equiv="refresh" content="0;url=${ogUrl}" />
</head>
<body>
  <a href="${ogUrl}">Tap to see who's coming →</a>
</body>
</html>`

  return new Response(html, {
    headers: {
      'Content-Type': 'text/html;charset=utf-8',
      'Cache-Control': 'public, max-age=300',
    },
  })
}
