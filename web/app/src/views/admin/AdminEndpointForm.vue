<template>
  <div class="container mx-auto px-4 py-8 max-w-5xl">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground dark:text-gray-100">{{ title }}</h1>
        <p v-if="isEdit" class="mt-1 font-mono text-sm text-muted-foreground dark:text-gray-400">
          {{ endpointKey }}<span v-if="version"> · version {{ version }}</span>
        </p>
      </div>
      <Button variant="outline" data-testid="admin-back" @click="goBack">Back</Button>
    </div>

    <div v-if="loading" class="py-12 flex justify-center"><Loading /></div>
    <template v-else>
      <div v-if="error" role="alert" data-testid="admin-error" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">
        <p class="whitespace-pre-line">{{ error }}</p>
        <Button v-if="versionConflict" variant="outline" size="sm" class="mt-2" data-testid="admin-reload" @click="reloadCurrentVersion">Reload current version</Button>
      </div>
      <div v-if="success" role="status" data-testid="admin-success" class="mb-4 border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200">{{ success }}</div>
      <div v-if="readOnly" class="mb-4 border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
        This endpoint is defined in the configuration file and can only be viewed.
      </div>
      <div v-if="exposure.length" role="status" data-testid="admin-endpoint-exposure" class="mb-4 border border-blue-300 bg-blue-50 px-4 py-3 text-sm text-blue-900 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-100">
        This endpoint will be publicly visible on the {{ exposure.length === 1 ? 'status page' : 'status pages' }}:
        <template v-for="(page, index) in exposure" :key="`${page.origin}-${page.slug}`">
          <a :href="`/status/${page.slug}`" target="_blank" rel="noopener" class="font-medium underline">{{ page.title }}</a>
          ({{ page.published ? 'published' : 'not published' }}, {{ page.reason === 'group' ? 'by group' : 'by key' }}){{ index < exposure.length - 1 ? ', ' : '' }}
        </template>
      </div>

      <div v-if="keyChange" role="status" data-testid="admin-key-change" class="mb-4 border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
        <p>
          Saving renames the key from <span class="font-mono">{{ endpointKey }}</span> to <span class="font-mono">{{ keyChange }}</span>.
          The history is kept, but the URLs of the badges and of the details page change.
        </p>
        <p v-if="receivesPush" class="mt-1" data-testid="admin-key-change-push">
          Push URLs with a global key use the new key; the push URL with the token of the endpoint does not change.
        </p>
        <p v-if="configPagesOfKey.length" class="mt-1" data-testid="admin-key-change-config-pages">
          These status pages of the configuration file select the endpoint by key and stop showing it until the file is updated:
          <template v-for="(page, index) in configPagesOfKey" :key="page.slug">
            <span class="font-medium">{{ page.title }}</span> ({{ page.slug }}){{ index < configPagesOfKey.length - 1 ? ', ' : '' }}
          </template>
        </p>
      </div>

      <div v-if="!readOnly" class="mb-4 flex border-b dark:border-gray-700" role="tablist">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          role="tab"
          :aria-selected="mode === tab.value"
          :data-testid="`admin-mode-${tab.value}`"
          :class="['-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors', mode === tab.value ? 'border-primary text-foreground dark:border-gray-100 dark:text-gray-100' : 'border-transparent text-muted-foreground hover:text-foreground dark:text-gray-400 dark:hover:text-gray-100']"
          @click="switchMode(tab.value)"
        >{{ tab.label }}</button>
      </div>

      <div v-if="mode === 'form' && !readOnly" class="space-y-6 border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="text-sm font-medium text-foreground dark:text-gray-200 sm:col-span-2">Monitor type
            <Select v-model="monitorType" :options="monitorTypeOptions" class="mt-1" data-testid="admin-field-type" />
            <p class="mt-1 text-xs font-normal text-muted-foreground dark:text-gray-400">
              <template v-if="isPush">Passive: Gatus does not check the endpoint, it receives pushes at its URL, like the Push monitors of the Uptime Kuma.</template>
              <template v-else-if="monitorType === 'dns'">The DNS query (query-name and query-type) is edited in YAML mode.</template>
              <template v-else>Active: Gatus checks the URL at every interval. The type follows the scheme of the URL.</template>
              <template v-if="isEdit"> Push endpoints and active endpoints cannot be converted into each other.</template>
            </p>
          </div>
          <label class="block text-sm font-medium text-foreground dark:text-gray-200">Name
            <Input v-model="form.name" class="mt-1 dark:border-gray-700" data-testid="admin-field-name" />
          </label>
          <div class="text-sm font-medium text-foreground dark:text-gray-200">Group
            <Select v-model="groupChoice" :options="groupChoiceOptions" placeholder="No group" class="mt-1" data-testid="admin-field-group-select" />
            <Input v-if="newGroup" v-model="form.group" placeholder="Name of the new group" class="mt-2 dark:border-gray-700" data-testid="admin-field-group" />
          </div>
          <template v-if="isPush">
            <label class="block text-sm font-medium text-foreground dark:text-gray-200">Heartbeat interval
              <Input v-model="form.heartbeatInterval" placeholder="1m (default)" class="mt-1 dark:border-gray-700" data-testid="admin-field-heartbeat" />
            </label>
            <label class="flex items-center gap-2 self-end pb-2 text-sm text-foreground dark:text-gray-200">
              <input v-model="form.enabled" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="admin-field-enabled" />
              Enabled
            </label>
          </template>
          <template v-else>
            <label class="block text-sm font-medium text-foreground dark:text-gray-200 sm:col-span-2">URL
              <Input v-model="form.url" :placeholder="urlPlaceholder" class="mt-1 font-mono dark:border-gray-700" data-testid="admin-field-url" />
            </label>
            <div v-if="monitorType === 'http'" class="text-sm font-medium text-foreground dark:text-gray-200">Method
              <Select v-model="form.method" :options="methodOptions" placeholder="GET (default)" class="mt-1" />
            </div>
            <label class="block text-sm font-medium text-foreground dark:text-gray-200">Interval
              <Input v-model="form.interval" placeholder="1m (default)" class="mt-1 dark:border-gray-700" data-testid="admin-field-interval" />
            </label>
            <div class="flex flex-wrap gap-x-6 gap-y-2 sm:col-span-2">
              <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200">
                <input v-model="form.enabled" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="admin-field-enabled" />
                Enabled
              </label>
              <template v-if="monitorType === 'http'">
                <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200" title="client.ignore-redirect">
                  <input v-model="form.followRedirects" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="admin-field-follow-redirects" />
                  Follow redirects
                </label>
                <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200" title="client.insecure: accepts self-signed, expired or incomplete certificate chains">
                  <input v-model="form.insecure" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="admin-field-insecure" />
                  Skip TLS certificate verification
                </label>
              </template>
              <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200" title="push: receive notifications at /api/push, in addition to the checks">
                <input v-model="form.acceptPush" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="admin-field-accept-push" />
                Accept push
              </label>
            </div>
          </template>
        </div>

        <section v-if="receivesPush" data-testid="admin-push-section">
          <h2 class="mb-1 text-sm font-semibold text-foreground dark:text-gray-200">Push</h2>
          <p class="mb-2 text-xs text-muted-foreground dark:text-gray-400">
            <template v-if="isPush">A failure is recorded for every heartbeat interval without push.</template>
            <template v-else>Pushes are recorded in the history of the endpoint, with its checks, and count for its uptime and alerts.</template>
          </p>
          <div class="flex flex-wrap items-end gap-2">
            <label class="block min-w-0 flex-1 text-sm text-foreground dark:text-gray-200">Token
              <Input
                v-model="form.token"
                :placeholder="isPush ? '' : 'Optional: without token, only the global push keys are accepted'"
                class="mt-1 font-mono dark:border-gray-700"
                data-testid="admin-field-push-token"
              />
            </label>
            <Button variant="outline" size="sm" data-testid="admin-generate-push-token" @click="form.token = generatePushToken()">Generate token</Button>
          </div>
          <p class="mt-1 text-xs text-muted-foreground dark:text-gray-400">To keep the scripts of an Uptime Kuma monitor, paste its token.</p>
          <div v-if="pushUrl" class="mt-3">
            <div class="text-sm text-foreground dark:text-gray-200">Push URL</div>
            <div class="mt-1 flex gap-2">
              <input :value="pushUrl" readonly class="h-10 w-full border border-input bg-background px-3 font-mono text-xs text-foreground dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100" data-testid="admin-push-url" @focus="$event.target.select()" />
              <Button variant="outline" size="sm" data-testid="admin-copy-push-url" @click="copyText(pushUrl)">Copy</Button>
            </div>
            <p class="mt-1 text-xs text-muted-foreground dark:text-gray-400">
              <template v-if="isPush">Call this URL at least every {{ form.heartbeatInterval.trim() || '1m' }}.</template>
              Optional parameters: <span class="font-mono">status</span> (<span class="font-mono">up</span>, or anything else for a failure), <span class="font-mono">msg</span> and <span class="font-mono">ping</span> (in milliseconds).
            </p>
            <pre class="mt-2 overflow-x-auto border bg-muted/50 p-2 font-mono text-xs text-foreground dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100" data-testid="admin-push-curl">curl -fsS "{{ pushUrl }}"</pre>
          </div>
          <p class="mt-2 text-xs text-muted-foreground dark:text-gray-400">
            With a global key of the Push keys tab: <span class="break-all font-mono">{{ globalKeyUrl }}</span>
          </p>
        </section>

        <template v-if="!isPush">
          <section>
            <h2 class="mb-2 text-sm font-semibold text-foreground dark:text-gray-200">Conditions</h2>
            <div v-for="(condition, index) in form.conditions" :key="`condition-${index}`" class="mb-2 flex gap-2">
              <Input v-model="form.conditions[index]" placeholder="[STATUS] == 200" class="font-mono dark:border-gray-700" :data-testid="`admin-field-condition-${index}`" />
              <Button variant="ghost" size="sm" aria-label="Remove condition" @click="form.conditions.splice(index, 1)">✕</Button>
            </div>
            <Button variant="outline" size="sm" data-testid="admin-add-condition" @click="form.conditions.push('')">Add condition</Button>
          </section>

          <section>
            <h2 class="mb-2 text-sm font-semibold text-foreground dark:text-gray-200">Headers</h2>
            <div v-for="(header, index) in form.headers" :key="`header-${index}`" class="mb-2 grid grid-cols-[1fr_2fr_auto] gap-2">
              <Input v-model="header.name" placeholder="Name" class="dark:border-gray-700" :data-testid="`admin-field-header-name-${index}`" />
              <Input v-model="header.value" placeholder="Value" class="font-mono dark:border-gray-700" :data-testid="`admin-field-header-value-${index}`" />
              <Button variant="ghost" size="sm" aria-label="Remove header" @click="form.headers.splice(index, 1)">✕</Button>
            </div>
            <Button variant="outline" size="sm" data-testid="admin-add-header" @click="form.headers.push({ name: '', value: '' })">Add header</Button>
          </section>
        </template>

        <section>
          <h2 class="mb-2 text-sm font-semibold text-foreground dark:text-gray-200">Alerts</h2>
          <p v-if="alertTypeOptions.length === 0" class="text-sm text-muted-foreground dark:text-gray-400">No alerting provider configured.</p>
          <div v-for="(alert, index) in form.alerts" :key="`alert-${index}`" class="mb-3 grid gap-2 border p-3 sm:grid-cols-2 dark:border-gray-700">
            <div class="text-sm text-foreground dark:text-gray-200">Type
              <Select v-model="alert.type" :options="alertTypeOptions" placeholder="Select" class="mt-1" />
            </div>
            <label class="block text-sm text-foreground dark:text-gray-200">Description
              <Input v-model="alert.description" class="mt-1 dark:border-gray-700" />
            </label>
            <label class="block text-sm text-foreground dark:text-gray-200">Failure threshold
              <Input v-model="alert.failureThreshold" type="number" min="1" placeholder="provider default" class="mt-1 dark:border-gray-700" />
            </label>
            <label class="block text-sm text-foreground dark:text-gray-200">Success threshold
              <Input v-model="alert.successThreshold" type="number" min="1" placeholder="provider default" class="mt-1 dark:border-gray-700" />
            </label>
            <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200">
              <input v-model="alert.sendOnResolved" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" />
              Send on resolved
            </label>
            <div class="text-right">
              <Button variant="ghost" size="sm" class="text-red-600 dark:text-red-400" @click="form.alerts.splice(index, 1)">Remove alert</Button>
            </div>
          </div>
          <Button v-if="alertTypeOptions.length > 0" variant="outline" size="sm" data-testid="admin-add-alert" @click="addAlert">Add alert</Button>
        </section>
      </div>

      <div v-else class="border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <textarea
          v-model="yamlText"
          :readonly="readOnly"
          spellcheck="false"
          data-testid="admin-yaml"
          class="min-h-[24rem] w-full border border-input bg-background p-3 font-mono text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100"
        ></textarea>
      </div>

      <div v-if="!readOnly" class="mt-6 flex flex-wrap gap-2">
        <Button variant="outline" :disabled="busy" data-testid="admin-validate" @click="validate">Validate</Button>
        <Button v-if="!isPush" variant="secondary" :disabled="busy" data-testid="admin-test" @click="test">Test</Button>
        <Button :disabled="busy" data-testid="admin-save" @click="save">Save</Button>
      </div>

      <div v-if="testResult" data-testid="admin-test-result" class="mt-6 border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <h2 class="mb-1 text-lg font-semibold text-foreground dark:text-gray-100">
          Test result:
          <span :class="testResult.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">{{ testResult.success ? 'success' : 'failure' }}</span>
        </h2>
        <p class="mb-3 text-sm text-muted-foreground dark:text-gray-400">
          Duration: {{ testResult.durationMs }} ms<span v-if="testResult.status"> · HTTP {{ testResult.status }}</span>
        </p>
        <ul class="space-y-1 font-mono text-sm">
          <li
            v-for="(condition, index) in testResult.conditionResults"
            :key="index"
            :class="condition.success ? 'text-green-700 dark:text-green-400' : 'text-red-700 dark:text-red-400'"
          >{{ condition.success ? '✓' : '✗' }} {{ condition.condition }}</li>
        </ul>
        <ul v-if="testResult.errors && testResult.errors.length" class="mt-3 list-disc pl-5 text-sm text-red-700 dark:text-red-400">
          <li v-for="(message, index) in testResult.errors" :key="index">{{ message }}</li>
        </ul>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import Loading from '@/components/Loading.vue'
import { adminApi, describeAdminError, jsonPayload, statusPagesApi, yamlPayload } from '@/utils/adminApi'
import { endpointKey as buildEndpointKey } from '@/utils/statusPage'
import { toYaml } from '@/utils/yaml'

const props = defineProps({
  endpointKey: { type: String, default: '' },
})

const router = useRouter()

const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'].map((method) => ({ label: method, value: method }))
const tabs = [
  { label: 'Form', value: 'form' },
  { label: 'YAML', value: 'yaml' },
]

// Monitor types of the form (fork): the active types follow the scheme of the URL, push endpoints are passive
const monitorTypes = [
  { label: 'HTTP(s)', value: 'http', placeholder: 'https://example.com/health', conditions: ['[STATUS] == 200'] },
  { label: 'TCP port', value: 'tcp', placeholder: 'tcp://example.com:443', conditions: ['[CONNECTED] == true'] },
  { label: 'Ping (ICMP)', value: 'icmp', placeholder: 'icmp://example.com', conditions: ['[CONNECTED] == true'] },
  { label: 'DNS', value: 'dns', placeholder: '8.8.8.8', conditions: ['[DNS_RCODE] == NOERROR'] },
  { label: 'SSH', value: 'ssh', placeholder: 'ssh://example.com:22', conditions: ['[CONNECTED] == true'] },
  { label: 'Push (passive)', value: 'push', placeholder: '', conditions: [] },
]
// Fields of the active endpoints, removed from the definition of a push endpoint
const activeFields = ['url', 'method', 'interval', 'conditions', 'headers', 'client', 'body', 'graphql', 'dns', 'ssh', 'ui', 'extra-labels', 'push']
const pushTokenAlphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
const maskedValue = '********'

const loading = ref(true)
const busy = ref(false)
const error = ref('')
const success = ref('')
const versionConflict = ref(false)
const testResult = ref(null)
const mode = ref('form')
const yamlText = ref('')
const version = ref(0)
const source = ref('admin')
const alertTypes = ref([])
// Groups of the existing endpoints, offered when creating an endpoint (fork)
const groupNames = ref([])
const newGroup = ref(false)
// Keys of the definition that the form does not edit (e.g. client, dns) are kept from this document
const baseDocument = ref({})
const monitorType = ref('http')

const emptyForm = () => ({
  name: '',
  group: '',
  url: '',
  method: '',
  interval: '',
  enabled: true,
  followRedirects: true,
  insecure: false,
  conditions: ['[STATUS] == 200'],
  headers: [],
  alerts: [],
  acceptPush: false,
  token: '',
  heartbeatInterval: '',
})

const form = reactive(emptyForm())

const isEdit = computed(() => props.endpointKey !== '')
const readOnly = computed(() => source.value === 'config')
const isPush = computed(() => monitorType.value === 'push')
const receivesPush = computed(() => isPush.value || form.acceptPush)
const title = computed(() => {
  if (!isEdit.value) {
    return 'New endpoint'
  }
  return readOnly.value ? 'Endpoint from the configuration file' : 'Edit endpoint'
})
const alertTypeOptions = computed(() => alertTypes.value.map((type) => ({ label: type, value: type })))
const monitorTypeOptions = computed(() => {
  // Push endpoints and active endpoints cannot be converted into each other
  const available = isEdit.value ? monitorTypes.filter((type) => (type.value === 'push') === isPush.value) : monitorTypes
  return available.map((type) => ({ label: type.label, value: type.value, testid: `admin-type-${type.value}` }))
})
const urlPlaceholder = computed(() => (monitorTypes.find((type) => type.value === monitorType.value) || monitorTypes[0]).placeholder)

const generatePushToken = () => {
  const characters = []
  while (characters.length < 32) {
    const bytes = new Uint8Array(64)
    window.crypto.getRandomValues(bytes)
    for (const byte of bytes) {
      // 248 is a multiple of the 62 characters, so that every character is equally likely
      if (byte < 248 && characters.length < 32) {
        characters.push(pushTokenAlphabet[byte % pushTokenAlphabet.length])
      }
    }
  }
  return characters.join('')
}

const currentKey = computed(() => (form.name.trim() ? buildEndpointKey(form.group.trim(), form.name.trim()) : ''))
const pushUrl = computed(() => {
  const token = form.token.trim()
  if (!token || token === maskedValue) {
    return ''
  }
  return `${window.location.origin}/api/push/${encodeURIComponent(token)}?status=up&msg=OK&ping=`
})
const globalKeyUrl = computed(() => `${window.location.origin}/api/push/<global-key>/${encodeURIComponent(currentKey.value || '<endpoint-key>')}?status=up&msg=OK&ping=`)

// Status pages of the configuration file that select the endpoint being edited by its current key
const configPagesOfKey = ref([])
// New key of a managed endpoint whose name or group changed in the form, or an empty string
const keyChange = computed(() => {
  if (!isEdit.value || readOnly.value || mode.value !== 'form' || !form.name.trim()) {
    return ''
  }
  return currentKey.value !== props.endpointKey ? currentKey.value : ''
})

// Groups are trimmed, so a value with a leading space never matches an existing group
const NEW_GROUP = ' new-group'
const groupChoiceOptions = computed(() => [
  { label: 'No group', value: '', testid: 'admin-group-none' },
  ...groupNames.value.map((name) => ({ label: name, value: name, testid: `admin-group-option-${name}` })),
  { label: 'New group…', value: NEW_GROUP, testid: 'admin-group-new' },
])
const groupChoice = computed({
  get: () => (newGroup.value ? NEW_GROUP : form.group),
  set: (value) => {
    newGroup.value = value === NEW_GROUP
    form.group = newGroup.value ? '' : value
  },
})

watch(monitorType, (type, previousType) => {
  if (isEdit.value) {
    return
  }
  if (type === 'push' && !form.token) {
    form.token = generatePushToken()
  }
  // The default conditions follow the type, unless they were changed
  const previous = monitorTypes.find((candidate) => candidate.value === previousType)
  const next = monitorTypes.find((candidate) => candidate.value === type)
  if (previous && next && next.conditions.length && JSON.stringify(form.conditions) === JSON.stringify(previous.conditions)) {
    form.conditions = [...next.conditions]
  }
})

watch(() => form.acceptPush, (enabled) => {
  // The token is optional for active endpoints: it can be cleared to accept only the global push keys
  if (enabled && !loading.value && !form.token) {
    form.token = generatePushToken()
  }
})

const loadGroupNames = async () => {
  const [endpoints, options] = await Promise.allSettled([adminApi.list(), statusPagesApi.options()])
  const names = new Set()
  if (endpoints.status === 'fulfilled') {
    (endpoints.value.data || []).forEach((item) => names.add((item.group || '').trim()))
  }
  if (options.status === 'fulfilled') {
    ((options.value.data && options.value.data.groups) || []).forEach((group) => names.add((group.name || '').trim()))
  }
  names.delete('')
  groupNames.value = [...names].sort((a, b) => a.localeCompare(b))
}

const clearMessages = () => {
  error.value = ''
  success.value = ''
  versionConflict.value = false
}

const monitorTypeOfDocument = (document) => {
  if (document.type === 'push') {
    return 'push'
  }
  const url = String(document.url || '').toLowerCase()
  for (const scheme of ['tcp', 'icmp', 'ssh']) {
    if (url.startsWith(`${scheme}://`)) {
      return scheme
    }
  }
  return document.dns ? 'dns' : 'http'
}

const formFromDocument = (document) => {
  const isPushDocument = document.type === 'push'
  Object.assign(form, {
    name: document.name || '',
    group: document.group || '',
    url: document.url || '',
    method: document.method || '',
    interval: document.interval || '',
    enabled: document.enabled !== false,
    followRedirects: !(document.client && document.client['ignore-redirect'] === true),
    insecure: Boolean(document.client && document.client.insecure === true),
    conditions: Array.isArray(document.conditions) ? document.conditions.map(String) : [],
    headers: document.headers ? Object.entries(document.headers).map(([name, value]) => ({ name, value: String(value) })) : [],
    alerts: Array.isArray(document.alerts)
      ? document.alerts.map((alert) => ({
        type: alert.type || '',
        description: alert.description || '',
        failureThreshold: alert['failure-threshold'] ? String(alert['failure-threshold']) : '',
        successThreshold: alert['success-threshold'] ? String(alert['success-threshold']) : '',
        sendOnResolved: alert['send-on-resolved'] === true,
        original: alert,
      }))
      : [],
    acceptPush: Boolean(document.push && document.push.enabled),
    token: String((isPushDocument ? document.token : document.push && document.push.token) || ''),
    heartbeatInterval: String((document.heartbeat && document.heartbeat.interval) || ''),
  })
  monitorType.value = monitorTypeOfDocument(document)
  newGroup.value = form.group !== '' && !groupNames.value.includes(form.group)
}

const setOrDelete = (document, key, value) => {
  if (value === '' || value === undefined || value === null) {
    delete document[key]
  } else {
    document[key] = value
  }
}

const documentFromForm = () => {
  const document = JSON.parse(JSON.stringify(baseDocument.value || {}))
  setOrDelete(document, 'name', form.name.trim())
  setOrDelete(document, 'group', form.group.trim())
  if (form.enabled) {
    delete document.enabled
  } else {
    document.enabled = false
  }
  if (isPush.value) {
    activeFields.forEach((field) => delete document[field])
    document.type = 'push'
    setOrDelete(document, 'token', form.token.trim())
    const heartbeat = { ...(document.heartbeat || {}) }
    setOrDelete(heartbeat, 'interval', form.heartbeatInterval.trim())
    setOrDelete(document, 'heartbeat', Object.keys(heartbeat).length ? heartbeat : null)
  } else {
    delete document.type
    delete document.token
    delete document.heartbeat
    setOrDelete(document, 'url', form.url.trim())
    setOrDelete(document, 'method', monitorType.value === 'http' ? form.method : null)
    setOrDelete(document, 'interval', form.interval.trim())
    // The other client options (e.g. oauth2, timeout) are only edited in YAML mode and kept as they are
    const client = { ...(document.client || {}) }
    setOrDelete(client, 'insecure', form.insecure ? true : null)
    setOrDelete(client, 'ignore-redirect', form.followRedirects ? null : true)
    setOrDelete(document, 'client', Object.keys(client).length ? client : null)
    const conditions = form.conditions.map((condition) => condition.trim()).filter(Boolean)
    setOrDelete(document, 'conditions', conditions.length ? conditions : null)
    const headers = {}
    form.headers.filter((header) => header.name.trim()).forEach((header) => {
      headers[header.name.trim()] = header.value
    })
    setOrDelete(document, 'headers', Object.keys(headers).length ? headers : null)
    if (form.acceptPush) {
      const push = { enabled: true }
      setOrDelete(push, 'token', form.token.trim())
      document.push = push
    } else {
      delete document.push
    }
  }
  const alerts = form.alerts.filter((alert) => alert.type).map((alert) => {
    const result = { ...(alert.original || {}), type: alert.type }
    setOrDelete(result, 'description', alert.description.trim())
    setOrDelete(result, 'failure-threshold', Number(alert.failureThreshold) > 0 ? Number(alert.failureThreshold) : null)
    setOrDelete(result, 'success-threshold', Number(alert.successThreshold) > 0 ? Number(alert.successThreshold) : null)
    setOrDelete(result, 'send-on-resolved', alert.sendOnResolved ? true : null)
    return result
  })
  setOrDelete(document, 'alerts', alerts.length ? alerts : null)
  return document
}

const currentPayload = () => (mode.value === 'form' ? jsonPayload(documentFromForm()) : yamlPayload(yamlText.value))

const addAlert = () => {
  form.alerts.push({ type: alertTypes.value[0] || '', description: '', failureThreshold: '', successThreshold: '', sendOnResolved: false, original: null })
}

const copyText = async (text) => {
  clearMessages()
  try {
    await navigator.clipboard.writeText(text)
    success.value = 'Copied to the clipboard.'
  } catch (e) {
    success.value = 'Select the text and copy it.'
  }
}

const loadDetail = async () => {
  const { data } = await adminApi.get(props.endpointKey)
  source.value = data.source
  version.value = data.version || 0
  baseDocument.value = data.definition.json || {}
  yamlText.value = data.definition.yaml || ''
  formFromDocument(baseDocument.value)
  // The token is masked in the definition and returned apart, to show the push URL
  if (data.pushToken) {
    form.token = data.pushToken
  }
  if (readOnly.value) {
    mode.value = 'yaml'
    return
  }
  try {
    const { data: exposed } = await statusPagesApi.exposure({ key: props.endpointKey })
    configPagesOfKey.value = ((exposed && exposed.statusPages) || []).filter((page) => page.origin === 'config' && page.reason === 'key')
  } catch (e) {
    configPagesOfKey.value = []
  }
}

const switchMode = async (target) => {
  if (target === mode.value) {
    return
  }
  clearMessages()
  if (target === 'yaml') {
    yamlText.value = `${toYaml(documentFromForm())}\n`
    mode.value = 'yaml'
    return
  }
  busy.value = true
  try {
    const { data } = await adminApi.parse(yamlPayload(yamlText.value))
    baseDocument.value = data.json || {}
    formFromDocument(baseDocument.value)
    mode.value = 'form'
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busy.value = false
  }
}

const validate = async () => {
  clearMessages()
  busy.value = true
  try {
    await adminApi.validate(currentPayload(), props.endpointKey)
    success.value = 'Valid definition.'
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busy.value = false
  }
}

const test = async () => {
  clearMessages()
  testResult.value = null
  busy.value = true
  try {
    const { data } = await adminApi.test(currentPayload(), props.endpointKey)
    testResult.value = data
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busy.value = false
  }
}

const save = async () => {
  clearMessages()
  busy.value = true
  try {
    if (isEdit.value) {
      const { data } = await adminApi.update(props.endpointKey, currentPayload(), version.value)
      const affected = (data && data.affectedConfigStatusPages) || []
      if (affected.length) {
        // Stays on the renamed endpoint, now under its new key, so that the affected pages can be read
        await router.replace({ name: 'AdminEndpointEdit', params: { endpointKey: data.key } })
        await loadDetail()
        success.value = `Saved as ${data.key}. Update the key in these status pages of the configuration file: ${affected.map((page) => `${page.title} (${page.slug})`).join(', ')}.`
        return
      }
    } else {
      await adminApi.create(currentPayload())
    }
    router.push({ name: 'AdminEndpoints' })
  } catch (e) {
    error.value = describeAdminError(e)
    versionConflict.value = e.status === 412
  } finally {
    busy.value = false
  }
}

const reloadCurrentVersion = async () => {
  clearMessages()
  try {
    await loadDetail()
    success.value = 'Current version loaded.'
  } catch (e) {
    error.value = describeAdminError(e)
  }
}

const goBack = () => {
  router.push({ name: 'AdminEndpoints' })
}

// Status pages on which the endpoint would appear, by group or by key (fork)
const exposure = ref([])
let exposureTimer = null

const refreshExposure = () => {
  clearTimeout(exposureTimer)
  exposureTimer = setTimeout(async () => {
    const group = form.group.trim()
    const name = form.name.trim()
    if (!group && !name) {
      exposure.value = []
      return
    }
    try {
      const { data } = await statusPagesApi.exposure({ group, key: name ? buildEndpointKey(group, name) : '' })
      exposure.value = (data && data.statusPages) || []
    } catch (e) {
      exposure.value = []
    }
  }, 400)
}

watch(() => [form.group, form.name], refreshExposure)

onUnmounted(() => clearTimeout(exposureTimer))

onMounted(async () => {
  // Loaded before the detail, so that its group is recognized as an existing one
  const groups = loadGroupNames()
  try {
    const { data } = await adminApi.metadata()
    alertTypes.value = (data && data.alertTypes) || []
  } catch (e) {
    alertTypes.value = []
  }
  await groups
  try {
    if (isEdit.value) {
      await loadDetail()
    }
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    loading.value = false
  }
})
</script>
