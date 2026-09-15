<template>
  <div class="container mx-auto px-4 py-8 max-w-5xl">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground dark:text-gray-100">{{ title }}</h1>
        <p v-if="isEdit" class="mt-1 font-mono text-sm text-muted-foreground dark:text-gray-400">
          <a :href="`/status/${slug}`" target="_blank" rel="noopener" class="underline-offset-4 hover:underline" data-testid="status-page-public-link">/status/{{ slug }}</a><span v-if="version"> · version {{ version }}</span>
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
        This status page is defined in the configuration file and can only be viewed.
      </div>
      <div v-if="savedError" role="alert" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">
        The saved definition is invalid and the page is not published: {{ savedError }}
      </div>

      <div v-if="readOnly" class="border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <pre class="whitespace-pre-wrap font-mono text-sm text-foreground dark:text-gray-100" data-testid="status-page-yaml">{{ yamlText }}</pre>
      </div>

      <div v-else class="space-y-4">
        <!-- General -->
        <section class="border bg-card p-5 dark:border-gray-700 dark:bg-gray-900" data-testid="status-page-section-general">
          <header class="mb-4">
            <h2 class="text-base font-semibold text-foreground dark:text-gray-100">General</h2>
            <p class="mt-0.5 text-xs text-muted-foreground dark:text-gray-400">Address, title and publication of the page.</p>
          </header>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="sm:col-span-2">
              <label for="status-page-slug" class="block text-sm font-medium text-foreground dark:text-gray-200">Slug</label>
              <div class="mt-1.5 flex gap-2">
                <div class="flex min-w-0 flex-1">
                  <span class="inline-flex h-10 shrink-0 items-center border border-r-0 border-input bg-muted/40 px-3 font-mono text-sm text-muted-foreground dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400">/status/</span>
                  <Input id="status-page-slug" v-model="form.slug" :disabled="isEdit" placeholder="infrastructure" class="min-w-0 font-mono dark:border-gray-700" data-testid="status-page-field-slug" />
                </div>
                <Button variant="outline" class="w-28 shrink-0" :disabled="!slugValid" data-testid="status-page-copy-link" @click="copyPublicUrl">{{ copied ? 'Copied' : 'Copy link' }}</Button>
                <a
                  v-if="isEdit"
                  :href="publicPath"
                  target="_blank"
                  rel="noopener"
                  class="inline-flex h-10 shrink-0 items-center border border-input bg-background px-4 text-sm font-medium hover:bg-accent dark:border-gray-700 dark:hover:bg-gray-800"
                  data-testid="status-page-open"
                >Open</a>
              </div>
              <p class="mt-1 text-xs text-muted-foreground dark:text-gray-400">Lowercase letters, digits and hyphens, up to 64 characters. It cannot be changed later.</p>
            </div>
            <label class="block sm:col-span-2">
              <span class="block text-sm font-medium text-foreground dark:text-gray-200">Title</span>
              <Input v-model="form.title" placeholder="Infrastructure" class="mt-1.5 dark:border-gray-700" data-testid="status-page-field-title" />
            </label>
            <label class="block sm:col-span-2">
              <span class="flex items-baseline justify-between text-sm font-medium text-foreground dark:text-gray-200">
                Description
                <span class="text-xs font-normal text-muted-foreground dark:text-gray-400">{{ form.description.length }}/1000</span>
              </span>
              <textarea
                v-model="form.description"
                rows="3"
                maxlength="1000"
                placeholder="Shown below the title of the page"
                class="mt-1.5 w-full border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:border-gray-700 dark:text-gray-100"
                data-testid="status-page-field-description"
              ></textarea>
            </label>
            <label :class="['flex items-start gap-3 border px-3 py-2.5 text-sm sm:col-span-2 dark:border-gray-700', form.enabled ? 'border-green-300 bg-green-50/60 dark:border-green-800 dark:bg-green-900/20' : '']">
              <input v-model="form.enabled" type="checkbox" class="mt-0.5 h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="status-page-field-enabled" />
              <span>
                <span class="block font-medium text-foreground dark:text-gray-200">Published</span>
                <span class="block text-xs text-muted-foreground dark:text-gray-400">
                  Visible without login at <span class="font-mono">{{ publicPath }}</span>. An unpublished page can still be validated and previewed here.
                </span>
              </span>
            </label>
            <label class="flex items-start gap-3 border px-3 py-2.5 text-sm sm:col-span-2 dark:border-gray-700">
              <input v-model="form.showCertificateExpiration" type="checkbox" class="mt-0.5 h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="status-page-field-show-certificate-expiration" />
              <span>
                <span class="block font-medium text-foreground dark:text-gray-200">Show certificate expiration</span>
                <span class="block text-xs text-muted-foreground dark:text-gray-400">Shows below the name of each endpoint how many days are left until its TLS certificate expires.</span>
              </span>
            </label>
          </div>
        </section>

        <!-- Groups -->
        <section class="border bg-card p-5 dark:border-gray-700 dark:bg-gray-900" data-testid="status-page-section-groups">
          <header class="mb-3 flex items-start justify-between gap-4">
            <div>
              <h2 class="text-base font-semibold text-foreground dark:text-gray-100">Groups</h2>
              <p class="mt-0.5 text-xs text-muted-foreground dark:text-gray-400">Every enabled endpoint of the group is shown on the page, including the ones created later.</p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span class="text-xs text-muted-foreground dark:text-gray-400" data-testid="status-page-groups-count">{{ form.groups.length }} selected</span>
              <Button v-if="form.groups.length" variant="ghost" size="sm" data-testid="status-page-groups-clear" @click="form.groups = []">Clear</Button>
            </div>
          </header>
          <p v-if="groupOptions.length === 0" class="text-sm text-muted-foreground dark:text-gray-400">No groups with endpoints.</p>
          <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            <label
              v-for="group in groupOptions"
              :key="group.name"
              :class="['flex cursor-pointer items-center gap-2 border px-3 py-2 text-sm hover:bg-accent/50 dark:border-gray-700 dark:hover:bg-gray-800/60', form.groups.includes(group.name) ? 'border-gray-900 dark:border-gray-300' : '']"
            >
              <input v-model="form.groups" type="checkbox" :value="group.name" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" :data-testid="`status-page-group-${group.name}`" />
              <span class="min-w-0 flex-1 truncate font-medium text-foreground dark:text-gray-100">{{ group.name }}</span>
              <span class="shrink-0 text-xs text-muted-foreground dark:text-gray-400">{{ group.endpoints === 1 ? '1 endpoint' : `${group.endpoints} endpoints` }}</span>
            </label>
          </div>
        </section>

        <!-- Endpoints -->
        <section class="border bg-card p-5 dark:border-gray-700 dark:bg-gray-900" data-testid="status-page-section-endpoints">
          <header class="mb-3 flex items-start justify-between gap-4">
            <div>
              <h2 class="text-base font-semibold text-foreground dark:text-gray-100">Endpoints</h2>
              <p class="mt-0.5 text-xs text-muted-foreground dark:text-gray-400">
                Picked one by one, in addition to the groups. <strong>Featured</strong> endpoints are shown at the top of the page with more details.
                Every endpoint of the page links to a public details page with its response time chart.
              </p>
            </div>
            <span class="shrink-0 text-xs text-muted-foreground dark:text-gray-400" data-testid="status-page-endpoints-count">
              {{ form.endpoints.length }} selected · {{ form.featured.length }}/{{ MAXIMUM_FEATURED }} featured
            </span>
          </header>
          <div class="mb-2 flex flex-wrap items-center gap-2">
            <Input v-model="endpointSearch" placeholder="Search by name, group or key" class="min-w-0 flex-1 dark:border-gray-700" data-testid="status-page-endpoint-search" />
            <label class="flex h-10 shrink-0 items-center gap-2 border px-3 text-sm text-foreground dark:border-gray-700 dark:text-gray-200">
              <input v-model="onlySelected" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="status-page-only-selected" />
              Only selected
            </label>
          </div>
          <div class="max-h-80 overflow-y-auto border dark:border-gray-700" data-testid="status-page-endpoint-list">
            <div class="sticky top-0 z-10 grid grid-cols-[1fr_auto] gap-4 bg-gray-50 px-3 py-2 text-xs font-medium text-muted-foreground shadow-[0_1px_0_0_rgb(229,231,235)] sm:grid-cols-[1fr_auto_auto] dark:bg-gray-800 dark:text-gray-400 dark:shadow-[0_1px_0_0_rgb(55,65,81)]">
              <span>Endpoint</span>
              <span class="hidden sm:block">Key</span>
              <span class="text-right">Featured</span>
            </div>
            <p v-if="filteredEndpoints.length === 0" class="px-3 py-4 text-sm text-muted-foreground dark:text-gray-400">No endpoints found.</p>
            <div
              v-for="endpoint in filteredEndpoints"
              :key="endpoint.key"
              :class="['grid grid-cols-[1fr_auto] items-center gap-4 border-t px-3 py-2 text-sm first-of-type:border-t-0 sm:grid-cols-[1fr_auto_auto] dark:border-gray-800', form.endpoints.includes(endpoint.key) ? 'bg-muted/40 dark:bg-gray-800/40' : '']"
            >
              <label class="flex min-w-0 cursor-pointer items-center gap-2">
                <input v-model="form.endpoints" type="checkbox" :value="endpoint.key" class="h-4 w-4 shrink-0 accent-gray-900 dark:accent-gray-100" :data-testid="`status-page-endpoint-${endpoint.key}`" />
                <span class="truncate font-medium text-foreground dark:text-gray-100">{{ endpoint.name }}</span>
                <span class="truncate text-muted-foreground dark:text-gray-400">{{ endpoint.group || 'no group' }}</span>
              </label>
              <span class="hidden font-mono text-xs text-muted-foreground sm:block dark:text-gray-500">{{ endpoint.key }}</span>
              <label class="flex items-center justify-end gap-1.5 text-xs text-foreground dark:text-gray-200" :title="featuredLimitReached(endpoint.key) ? `At most ${MAXIMUM_FEATURED} featured endpoints` : ''">
                <input
                  v-model="form.featured"
                  type="checkbox"
                  :value="endpoint.key"
                  :disabled="featuredLimitReached(endpoint.key)"
                  class="h-4 w-4 accent-gray-900 dark:accent-gray-100"
                  :data-testid="`status-page-featured-${endpoint.key}`"
                />
                <span class="sm:sr-only">Featured</span>
              </label>
            </div>
          </div>
        </section>
      </div>

      <div class="mt-6 flex flex-wrap items-center gap-2 border-t pt-4 dark:border-gray-700">
        <Button v-if="!readOnly" variant="outline" :disabled="busy" data-testid="status-page-validate" @click="validate">Validate</Button>
        <Button variant="secondary" :disabled="busy" data-testid="status-page-preview-button" @click="showPreview">Preview</Button>
        <Button v-if="!readOnly" :disabled="busy" class="sm:ml-auto" data-testid="status-page-save" @click="save">Save</Button>
      </div>

      <div v-if="validation" data-testid="status-page-validation" class="mt-6 border bg-card p-5 text-sm dark:border-gray-700 dark:bg-gray-900">
        <h2 class="text-base font-semibold text-foreground dark:text-gray-100">Validation</h2>
        <p class="mt-1 text-foreground dark:text-gray-100">The page will show {{ validation.endpoints === 1 ? '1 endpoint' : `${validation.endpoints} endpoints` }}.</p>
        <ul v-if="validation.warnings.length" class="mt-2 list-disc pl-5 text-amber-800 dark:text-amber-300">
          <li v-for="warning in validation.warnings" :key="`${warning.type}-${warning.value}`">
            {{ warningMessage(warning) }}
          </li>
        </ul>
      </div>

      <section v-if="preview" data-testid="status-page-preview" class="mt-6 border bg-card dark:border-gray-700 dark:bg-gray-900">
        <button
          type="button"
          :aria-expanded="previewExpanded"
          data-testid="status-page-preview-toggle"
          class="flex w-full items-center gap-3 p-5 text-left hover:bg-accent/50 dark:hover:bg-gray-800/60"
          @click="previewExpanded = !previewExpanded"
        >
          <ChevronDown v-if="previewExpanded" class="h-4 w-4 shrink-0 text-muted-foreground dark:text-gray-400" />
          <ChevronRight v-else class="h-4 w-4 shrink-0 text-muted-foreground dark:text-gray-400" />
          <span class="min-w-0 flex-1">
            <span class="block text-base font-semibold text-foreground dark:text-gray-100">Preview of the saved version</span>
            <span class="mt-0.5 block text-xs text-muted-foreground dark:text-gray-400">{{ preview.title }} · {{ pageStatusLabel(preview.status) }}</span>
          </span>
        </button>
        <div v-if="previewExpanded" class="border-t px-5 pb-5 pt-4 dark:border-gray-700">
          <p v-if="preview.groups.length === 0 && (preview.featured || []).length === 0" class="text-sm text-muted-foreground dark:text-gray-400">No endpoints selected.</p>
          <div v-if="(preview.featured || []).length" class="mb-4" data-testid="status-page-preview-featured">
            <h3 class="border-b pb-1 text-sm font-semibold text-foreground dark:border-gray-800 dark:text-gray-100">Featured</h3>
            <ul class="mt-1 space-y-1 text-sm">
              <li v-for="endpoint in preview.featured" :key="`${endpoint.group}-${endpoint.name}`" class="flex justify-between gap-4">
                <span class="text-foreground dark:text-gray-200">{{ endpoint.name }} <span class="text-muted-foreground dark:text-gray-400">{{ endpoint.group }}</span></span>
                <span class="text-muted-foreground dark:text-gray-400">{{ endpointStatusLabel(endpoint.status) }} · 24h {{ formatUptime(endpoint.uptime['24h']) }}</span>
              </li>
            </ul>
          </div>
          <div v-for="group in preview.groups" :key="group.name || '__without-group__'" class="mb-4 last:mb-0">
            <h3 class="border-b pb-1 text-sm font-semibold text-foreground dark:border-gray-800 dark:text-gray-100">{{ group.name || 'Other services' }}</h3>
            <ul class="mt-1 space-y-1 text-sm">
              <li v-for="endpoint in group.endpoints" :key="endpoint.name" class="flex justify-between gap-4">
                <span class="text-foreground dark:text-gray-200">{{ endpoint.name }}</span>
                <span class="text-muted-foreground dark:text-gray-400">{{ endpointStatusLabel(endpoint.status) }} · 24h {{ formatUptime(endpoint.uptime['24h']) }}</span>
              </li>
            </ul>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ChevronDown, ChevronRight } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Loading from '@/components/Loading.vue'
import { describeStatusPageError, statusPagesApi } from '@/utils/adminApi'
import { formatUptime, SLUG_PATTERN, STATUS_LABELS } from '@/utils/statusPage'

const props = defineProps({
  slug: { type: String, default: '' }
})

const router = useRouter()

const loading = ref(true)
const busy = ref(false)
const error = ref('')
const success = ref('')
const versionConflict = ref(false)
const origin = ref('admin')
const version = ref(0)
const yamlText = ref('')
const savedError = ref('')
const options = ref({ groups: [], endpoints: [] })
const endpointSearch = ref('')
// Shows only the selected and featured endpoints in the list (fork)
const onlySelected = ref(false)
const validation = ref(null)
const preview = ref(null)
const previewExpanded = ref(true)
const copied = ref(false)
let copiedTimer = null

const MAXIMUM_FEATURED = 10

// The deprecated charts are not part of the form: saving a page removes them
const emptyForm = () => ({ slug: '', title: '', description: '', enabled: false, groups: [], endpoints: [], featured: [], showCertificateExpiration: false })
const form = reactive(emptyForm())

// Message shown after the route changes from the creation to the edition of the created page
let pendingSuccess = ''

const isEdit = computed(() => props.slug !== '')
const readOnly = computed(() => origin.value === 'config')
const slugValid = computed(() => SLUG_PATTERN.test(form.slug.trim()))
const publicPath = computed(() => `/status/${form.slug.trim() || '<slug>'}`)
const title = computed(() => {
  if (!isEdit.value) {
    return 'New status page'
  }
  return readOnly.value ? 'Status page from the configuration file' : 'Edit status page'
})

// Groups of the options, plus the selected groups that have no endpoint at the moment
const groupOptions = computed(() => {
  const names = new Set(options.value.groups.map((group) => group.name))
  const missing = form.groups.filter((name) => !names.has(name)).map((name) => ({ name, endpoints: 0 }))
  return [...options.value.groups, ...missing]
})

// Endpoints of the options matching the search, plus the selected and featured keys that do not exist at the moment
const filteredEndpoints = computed(() => {
  const keys = new Set(options.value.endpoints.map((endpoint) => endpoint.key))
  const missing = [...new Set([...form.endpoints, ...form.featured])].filter((key) => !keys.has(key)).map((key) => ({ key, name: key, group: '' }))
  const query = endpointSearch.value.trim().toLowerCase()
  return [...options.value.endpoints, ...missing].filter((endpoint) =>
    (!onlySelected.value || form.endpoints.includes(endpoint.key) || form.featured.includes(endpoint.key)) &&
    (!query || [endpoint.key, endpoint.name, endpoint.group].some((value) => (value || '').toLowerCase().includes(query)))
  )
})

const featuredLimitReached = (key) => !form.featured.includes(key) && form.featured.length >= MAXIMUM_FEATURED

const warningMessage = (warning) => ({
  group: `Group ${warning.value} has no endpoints at the moment.`,
  endpoint: `Endpoint ${warning.value} does not exist at the moment.`,
  featured: `Featured endpoint ${warning.value} does not exist at the moment.`,
  charts: `The saved charts (${warning.value}) are no longer used: every endpoint of the page has a details page with its response time chart. Saving the page removes them.`
}[warning.type] || `${warning.type}: ${warning.value}`)

const pageStatusLabel = (status) => STATUS_LABELS[status]?.page || STATUS_LABELS.unknown.page
const endpointStatusLabel = (status) => STATUS_LABELS[status]?.endpoint || STATUS_LABELS.unknown.endpoint

const clearMessages = () => {
  error.value = ''
  success.value = ''
  versionConflict.value = false
}

const copyPublicUrl = async () => {
  clearMessages()
  const url = `${window.location.origin}/status/${form.slug.trim()}`
  try {
    await navigator.clipboard.writeText(url)
    copied.value = true
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => { copied.value = false }, 2000)
  } catch (e) {
    success.value = `The browser did not allow copying: the address of the page is ${url}`
  }
}

const currentDocument = () => ({
  slug: form.slug.trim(),
  title: form.title.trim(),
  description: form.description.trim(),
  groups: [...form.groups],
  endpoints: [...form.endpoints],
  featured: [...form.featured],
  // Only sent when checked, like the other optional fields of the definition (fork)
  ...(form.showCertificateExpiration ? { 'show-certificate-expiration': true } : {}),
  enabled: form.enabled
})

const loadDetail = async () => {
  const { data } = await statusPagesApi.get(props.slug)
  origin.value = data.origin
  version.value = data.version || 0
  yamlText.value = data.yaml || ''
  savedError.value = data.error || ''
  const definition = data.definition
  Object.assign(form, emptyForm(), { slug: data.slug })
  if (definition) {
    Object.assign(form, {
      title: definition.title || '',
      description: definition.description || '',
      enabled: data.origin === 'config' ? definition.enabled !== false : definition.enabled === true,
      groups: definition.groups || [],
      endpoints: definition.endpoints || [],
      featured: definition.featured || [],
      showCertificateExpiration: definition['show-certificate-expiration'] === true
    })
  }
}

const load = async () => {
  loading.value = true
  clearMessages()
  validation.value = null
  preview.value = null
  try {
    const { data } = await statusPagesApi.options()
    options.value = data || { groups: [], endpoints: [] }
    if (isEdit.value) {
      await loadDetail()
    } else {
      origin.value = 'admin'
      version.value = 0
      savedError.value = ''
      Object.assign(form, emptyForm())
    }
  } catch (e) {
    error.value = describeStatusPageError(e)
  } finally {
    loading.value = false
  }
}

const validate = async () => {
  clearMessages()
  busy.value = true
  try {
    const { data } = await statusPagesApi.validate(currentDocument(), isEdit.value ? props.slug : '')
    validation.value = data
    success.value = 'Valid definition.'
  } catch (e) {
    validation.value = null
    error.value = describeStatusPageError(e)
  } finally {
    busy.value = false
  }
}

const showPreview = async () => {
  clearMessages()
  if (!isEdit.value) {
    await validate()
    if (!error.value) {
      success.value = 'Valid definition. Save the page to preview it with the data of its endpoints.'
    }
    return
  }
  busy.value = true
  try {
    const { data } = await statusPagesApi.preview(props.slug)
    preview.value = data
    previewExpanded.value = true
  } catch (e) {
    error.value = describeStatusPageError(e)
  } finally {
    busy.value = false
  }
}

const save = async () => {
  clearMessages()
  busy.value = true
  try {
    if (isEdit.value) {
      const { data } = await statusPagesApi.update(props.slug, currentDocument(), version.value)
      version.value = data.version || version.value
      savedError.value = ''
      preview.value = null
      success.value = data.published ? 'Status page saved and published.' : 'Status page saved. It is not published.'
    } else {
      const { data } = await statusPagesApi.create(currentDocument())
      pendingSuccess = data.published ? 'Status page created and published.' : 'Status page created. It is not published yet: check "Published" once you have reviewed the preview.'
      await router.push({ name: 'AdminStatusPageEdit', params: { slug: data.slug } })
    }
  } catch (e) {
    error.value = describeStatusPageError(e)
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
    error.value = describeStatusPageError(e)
  }
}

const goBack = () => {
  router.push({ name: 'AdminStatusPages' })
}

// The same component serves the creation and the edition: after a creation, the route changes to the edition
watch(() => props.slug, async (slug, previousSlug) => {
  if (slug === previousSlug) {
    return
  }
  await load()
  success.value = pendingSuccess
  pendingSuccess = ''
})

onMounted(load)
onUnmounted(() => clearTimeout(copiedTimer))
</script>
