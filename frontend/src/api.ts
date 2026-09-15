// Hostname-based rather than VITE_API_URL: a blank Vercel env var once
// overrode the .env file at build time and pointed production at nothing.
export const API =
  window.location.hostname === 'localhost'
    ? 'http://localhost:8080'
    : 'https://ollae-backend.fly.dev'

const TIMEOUT_MS = 15_000

// The Fly machine scales to zero; a cold start that never answers should
// surface as an error rather than a spinner that spins forever.
export function api(path: string, init?: RequestInit) {
  return fetch(`${API}${path}`, { signal: AbortSignal.timeout(TIMEOUT_MS), ...init })
}
