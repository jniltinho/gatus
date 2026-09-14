<template>
  <div class="container mx-auto px-4 py-8 max-w-5xl">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground dark:text-gray-100">{{ title }}</h1>
        <p v-if="isEdit" class="mt-1 font-mono text-sm text-muted-foreground dark:text-gray-400">
          /status/{{ slug }}<span v-if="version"> · version {{ version }}</span>
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

      <div v-else class="space-y-6 border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block text-sm font-medium text-foreground dark:text-gray-200">Slug
            <Input v-model="form.slug" :disabled="isEdit" placeholder="infrastructure" class="mt-1 font-mono dark:border-gray-700" data-testid="status-page-field-slug" />
            <span class="mt-1 block text-xs font-normal text-muted-foreground dark:text-gray-400">Lowercase letters, digits and hyphens. It cannot be changed later.</span>
          </label>
          <label class="block text-sm font-medium text-foreground dark:text-gray-200">Title
            <Input v-model="form.title" placeholder="Infrastructure" class="mt-1 dark:border-gray-700" data-testid="status-page-field-title" />
          </label>
          <label class="block text-sm font-medium text-foreground dark:text-gray-200 sm:col-span-2">Description
            <textarea
              v-model="form.description"
              rows="2"
              maxlength="1000"
              class="mt-1 w-full border border-input bg-background px-3 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:border-gray-700 dark:text-gray-100"
              data-testid="status-page-field-description"
            ></textarea>
          </label>
          <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200 sm:col-span-2">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="status-page-field-enabled" />
            Published: visible without login at {{ publicPath }}
          </label>
        </div>

        <section>
          <h2 class="mb-1 text-sm font-semibold text-foreground dark:text-gray-200">Groups</h2>
          <p class="mb-2 text-xs text-muted-foreground dark:text-gray-400">Every enabled endpoint of the group is shown on the page, including the ones created later.</p>
          <p v-if="groupOptions.length === 0" class="text-sm text-muted-foreground dark:text-gray-400">No groups with endpoints.</p>
          <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            <label v-for="group in groupOptions" :key="group.name" class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200">
              <input v-model="form.groups" type="checkbox" :value="group.name" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" :data-testid="`status-page-group-${group.name}`" />
              <span>{{ group.name }}</span>
              <span class="text-xs text-muted-foreground dark:text-gray-400">{{ group.endpoints === 1 ? '1 endpoint' : `${group.endpoints} endpoints` }}</span>
            </label>
          </div>
        </section>

        <section>
          <h2 class="mb-1 text-sm font-semibold text-foreground dark:text-gray-200">Endpoints</h2>
          <p class="mb-2 text-xs text-muted-foreground dark:text-gray-400">Endpoints picked one by one, in addition to the groups. {{ form.endpoints.length }} selected.</p>
          <Input v-model="endpointSearch" placeholder="Search by name, group or key" class="mb-2 dark:border-gray-700" data-testid="status-page-endpoint-search" />
          <div class="max-h-72 overflow-y-auto border dark:border-gray-700">
            <p v-if="filteredEndpoints.length === 0" class="px-3 py-4 text-sm text-muted-foreground dark:text-gray-400">No endpoints found.</p>
            <label v-for="endpoint in filteredEndpoints" :key="endpoint.key" class="flex items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0 dark:border-gray-800">
              <input v-model="form.endpoints" type="checkbox" :value="endpoint.key" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" :data-testid="`status-page-endpoint-${endpoint.key}`" />
              <span class="font-medium text-foreground dark:text-gray-100">{{ endpoint.name }}</span>
              <span class="text-muted-foreground dark:text-gray-400">{{ endpoint.group || 'no group' }}</span>
              <span class="ml-auto font-mono text-xs text-muted-foreground dark:text-gray-500">{{ endpoint.key }}</span>
            </label>
          </div>
        </section>
      </div>

      <div class="mt-6 flex flex-wrap gap-2">
        <Button v-if="!readOnly" variant="outline" :disabled="busy" data-testid="status-page-validate" @click="validate">Validate</Button>
        <Button variant="secondary" :disabled="busy" data-testid="status-page-preview-button" @click="showPreview">Preview</Button>
        <Button v-if="!readOnly" :disabled="busy" data-testid="status-page-save" @click="save">Save</Button>
      </div>

      <div v-if="validation" data-testid="status-page-validation" class="mt-6 border bg-card p-6 text-sm dark:border-gray-700 dark:bg-gray-900">
        <p class="text-foreground dark:text-gray-100">The page will show {{ validation.endpoints === 1 ? '1 endpoint' : `${validation.endpoints} endpoints` }}.</p>
        <ul v-if="validation.warnings.length" class="mt-2 list-disc pl-5 text-amber-800 dark:text-amber-300">
          <li v-for="warning in validation.warnings" :key="`${warning.type}-${warning.value}`">
            {{ warning.type === 'group' ? `Group ${warning.value} has no endpoints at the moment.` : `Endpoint ${warning.value} does not exist at the moment.` }}
          </li>
        </ul>
      </div>

      <div v-if="preview" data-testid="status-page-preview" class="mt-6 border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <h2 class="text-lg font-semibold text-foreground dark:text-gray-100">Preview of the saved version</h2>
        <p class="mb-4 text-sm text-muted-foreground dark:text-gray-400">{{ preview.title }} · {{ pageStatusLabel(preview.status) }}</p>
        <p v-if="preview.groups.length === 0" class="text-sm text-muted-foreground dark:text-gray-400">No endpoints selected.</p>
        <div v-for="group in preview.groups" :key="group.name || '__without-group__'" class="mb-4">
          <h3 class="border-b pb-1 text-sm font-semibold text-foreground dark:border-gray-800 dark:text-gray-100">{{ group.name || 'Other services' }}</h3>
          <ul class="mt-1 space-y-1 text-sm">
            <li v-for="endpoint in group.endpoints" :key="endpoint.name" class="flex justify-between gap-4">
              <span class="text-foreground dark:text-gray-200">{{ endpoint.name }}</span>
              <span class="text-muted-foreground dark:text-gray-400">{{ endpointStatusLabel(endpoint.status) }} · 24h {{ formatUptime(endpoint.uptime['24h']) }}</span>
            </li>
          </ul>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Loading from '@/components/Loading.vue'
import { describeStatusPageError, statusPagesApi } from '@/utils/adminApi'
import { formatUptime, STATUS_LABELS } from '@/utils/statusPage'

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
const validation = ref(null)
const preview = ref(null)

const emptyForm = () => ({ slug: '', title: '', description: '', enabled: false, groups: [], endpoints: [] })
const form = reactive(emptyForm())

// Message shown after the route changes from the creation to the edition of the created page
let pendingSuccess = ''

const isEdit = computed(() => props.slug !== '')
const readOnly = computed(() => origin.value === 'config')
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

// Endpoints of the options matching the search, plus the selected keys that do not exist at the moment
const filteredEndpoints = computed(() => {
  const keys = new Set(options.value.endpoints.map((endpoint) => endpoint.key))
  const missing = form.endpoints.filter((key) => !keys.has(key)).map((key) => ({ key, name: key, group: '' }))
  const query = endpointSearch.value.trim().toLowerCase()
  return [...options.value.endpoints, ...missing].filter((endpoint) =>
    !query || [endpoint.key, endpoint.name, endpoint.group].some((value) => (value || '').toLowerCase().includes(query))
  )
})

const pageStatusLabel = (status) => STATUS_LABELS[status]?.page || STATUS_LABELS.unknown.page
const endpointStatusLabel = (status) => STATUS_LABELS[status]?.endpoint || STATUS_LABELS.unknown.endpoint

const clearMessages = () => {
  error.value = ''
  success.value = ''
  versionConflict.value = false
}

const currentDocument = () => ({
  slug: form.slug.trim(),
  title: form.title.trim(),
  description: form.description.trim(),
  groups: [...form.groups],
  endpoints: [...form.endpoints],
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
      endpoints: definition.endpoints || []
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
</script>
