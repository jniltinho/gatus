// Helpers of the public status pages (fork)

export const SLUG_PATTERN = /^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$/

const sanitizeKeyPart = (value) => (value || '').toLowerCase().trim().replace(/[/_., #+&]/g, '-')

// endpointKey mirrors key.ConvertGroupAndNameToKey of the backend
export const endpointKey = (group, name) => `${sanitizeKeyPart(group)}_${sanitizeKeyPart(name)}`

// Labels of the statuses of the payload, for a page ("page"), a group ("group") and an endpoint ("endpoint")
export const STATUS_LABELS = {
  operational: { page: 'All systems operational', group: 'Operational' },
  degraded: { page: 'Partial outage', group: 'Partial outage' },
  down: { page: 'Major outage', group: 'Major outage', endpoint: 'Down' },
  up: { endpoint: 'Up' },
  unknown: { page: 'No data', group: 'No data', endpoint: 'No data' }
}

// Like the dashboard, numbers and dates use the locale of the browser
const percentFormat = new Intl.NumberFormat(undefined, { style: 'percent', maximumFractionDigits: 2 })
const dateTimeFormat = new Intl.DateTimeFormat(undefined, { dateStyle: 'short', timeStyle: 'medium' })

// formatUptime formats an uptime between 0 and 1, or a dash without execution during the period
export const formatUptime = (uptime) => (uptime === null || uptime === undefined ? '—' : percentFormat.format(uptime))

export const formatDateTime = (timestamp) => dateTimeFormat.format(new Date(timestamp))

// relativeTimeLabel describes how long ago the timestamp was, never in the future when the clocks differ
export const relativeTimeLabel = (timestamp, now) => {
  const seconds = Math.max(0, Math.round((now - Date.parse(timestamp)) / 1000))
  if (seconds < 10) {
    return 'just now'
  }
  if (seconds < 60) {
    return `${seconds} seconds ago`
  }
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) {
    return minutes === 1 ? '1 minute ago' : `${minutes} minutes ago`
  }
  const hours = Math.floor(minutes / 60)
  return hours === 1 ? '1 hour ago' : `${hours} hours ago`
}

// Periods of the response time charts, like the endpoint details page of the dashboard
export const RESPONSE_TIME_DURATIONS = [
  { value: '24h', label: '24 hours' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' }
]

// The server caches the response time charts for 5 minutes
const RESPONSE_TIMES_TTL_MS = 5 * 60 * 1000

// formatMilliseconds formats a response time, or a dash without execution
export const formatMilliseconds = (milliseconds) => (milliseconds === null || milliseconds === undefined ? '—' : `${milliseconds} ms`)

// createResponseTimesLoader returns a function that loads the response time charts of a status page for a duration,
// sharing one request per duration between all the charts of the page
export const createResponseTimesLoader = (slug) => {
  const requests = new Map()
  return (duration) => {
    const cached = requests.get(duration)
    if (cached && Date.now() - cached.time < RESPONSE_TIMES_TTL_MS) {
      return cached.promise
    }
    const promise = fetch(`/api/v1/status-pages/${encodeURIComponent(slug)}/response-times/${encodeURIComponent(duration)}`, { credentials: 'omit' })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`unexpected status ${response.status}`)
        }
        return response.json()
      })
    promise.catch(() => requests.delete(duration))
    requests.set(duration, { promise, time: Date.now() })
    return promise
  }
}
