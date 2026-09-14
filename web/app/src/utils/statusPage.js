// Helpers of the public status pages (fork)

export const SLUG_PATTERN = /^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$/

const sanitizeKeyPart = (value) => (value || '').toLowerCase().trim().replace(/[/_., #+&]/g, '-')

// endpointKey mirrors key.ConvertGroupAndNameToKey of the backend
export const endpointKey = (group, name) => `${sanitizeKeyPart(group)}_${sanitizeKeyPart(name)}`

// Labels of the statuses of the payload, for a page or group ("page") and for an endpoint ("endpoint")
export const STATUS_LABELS = {
  operational: { page: 'Todos os sistemas operacionais', group: 'Operacional' },
  degraded: { page: 'Degradação parcial', group: 'Degradação parcial' },
  down: { page: 'Indisponível', group: 'Indisponível', endpoint: 'Fora do ar' },
  up: { endpoint: 'No ar' },
  unknown: { page: 'Sem dados', group: 'Sem dados', endpoint: 'Sem dados' }
}

const percentFormat = new Intl.NumberFormat('pt-BR', { style: 'percent', maximumFractionDigits: 2 })
const dateTimeFormat = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short', timeStyle: 'medium' })

// formatUptime formats an uptime between 0 and 1, or a dash without execution during the period
export const formatUptime = (uptime) => (uptime === null || uptime === undefined ? '—' : percentFormat.format(uptime))

export const formatDateTime = (timestamp) => dateTimeFormat.format(new Date(timestamp))

// relativeTimeLabel describes how long ago the timestamp was, never in the future when the clocks differ
export const relativeTimeLabel = (timestamp, now) => {
  const seconds = Math.max(0, Math.round((now - Date.parse(timestamp)) / 1000))
  if (seconds < 10) {
    return 'agora mesmo'
  }
  if (seconds < 60) {
    return `há ${seconds} segundos`
  }
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) {
    return minutes === 1 ? 'há 1 minuto' : `há ${minutes} minutos`
  }
  const hours = Math.floor(minutes / 60)
  return hours === 1 ? 'há 1 hora' : `há ${hours} horas`
}
