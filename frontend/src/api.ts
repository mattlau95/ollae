// Hostname-based rather than VITE_API_URL: a blank Vercel env var once
// overrode the .env file at build time and pointed production at nothing.
export const API =
  window.location.hostname === 'localhost'
    ? 'http://localhost:8080'
    : 'https://ollae-backend.fly.dev'
