// Client of the administration API of endpoints (/api/v1/admin)
const BASE_URL = '/api/v1/admin'

export class AdminApiError extends Error {
  constructor(status, message) {
    super(message)
    this.status = status
  }
}

const encodeKey = (key) => encodeURIComponent(key)

const withKey = (path, key) => (key ? `${path}?key=${encodeKey(key)}` : path)

async function request(method, path, { body, contentType, version } = {}) {
  const headers = {}
  if (body !== undefined) {
    headers['Content-Type'] = contentType
  }
  if (version) {
    headers['If-Match'] = `"${version}"`
  }
  const response = await fetch(BASE_URL + path, { method, headers, body, credentials: 'include' })
  const text = await response.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch (e) {
      data = { error: text }
    }
  }
  if (!response.ok) {
    throw new AdminApiError(response.status, (data && data.error) || response.statusText)
  }
  return { data, status: response.status }
}

export const jsonPayload = (document) => ({ body: JSON.stringify(document), contentType: 'application/json' })

export const yamlPayload = (text) => ({ body: text, contentType: 'application/yaml' })

export const adminApi = {
  metadata: () => request('GET', '/metadata'),
  list: () => request('GET', '/endpoints'),
  get: (key) => request('GET', `/endpoints/${encodeKey(key)}`),
  parse: (payload) => request('POST', '/endpoints/parse', payload),
  validate: (payload, key) => request('POST', withKey('/endpoints/validate', key), payload),
  test: (payload, key) => request('POST', withKey('/endpoints/test', key), payload),
  create: (payload) => request('POST', '/endpoints', payload),
  update: (key, payload, version) => request('PUT', `/endpoints/${encodeKey(key)}`, { ...payload, version }),
  setEnabled: (key, enabled, version) => request('POST', `/endpoints/${encodeKey(key)}/${enabled ? 'enable' : 'disable'}`, { version }),
  remove: (key, version) => request('DELETE', `/endpoints/${encodeKey(key)}`, { version }),
}

export function describeAdminError(error) {
  switch (error && error.status) {
    case 401:
      return 'Autenticação necessária.'
    case 403:
      return 'Sem permissão de administrador.'
    case 412:
      return 'O endpoint foi alterado por outra pessoa desde que você o abriu.'
    case 429:
      return 'Muitos testes em andamento. Tente novamente em instantes.'
    case 503:
      return 'O Gatus está iniciando ou recarregando a configuração. Tente novamente em instantes.'
    default:
      return (error && error.message) || 'Erro inesperado.'
  }
}
