<template>
  <div class="container mx-auto px-4 py-8 max-w-5xl">
    <div v-if="state === 'loading'" class="py-16 flex justify-center"><Loading /></div>

    <section v-else-if="state === 'not-found'" class="py-16 text-center" data-testid="status-page-not-found">
      <h1 class="text-2xl font-bold tracking-tight">Page not found</h1>
      <p class="mt-2 text-muted-foreground">Check the address of the status page.</p>
    </section>

    <template v-else>
      <div
        v-if="errorMessage"
        role="alert"
        data-testid="status-page-error"
        :class="['mb-4 border px-4 py-3 text-sm', page ? 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100' : 'border-red-300 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200']"
      >
        {{ errorMessage }}
      </div>

      <template v-if="page">
        <header class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight" data-testid="status-page-title">{{ page.title }}</h1>
          <p v-if="page.description" class="mt-2 text-muted-foreground whitespace-pre-line" data-testid="status-page-description">{{ page.description }}</p>
        </header>

        <StatusSummary :status="page.status" :updated-at="page.updatedAt" :now="now" />

        <p v-if="page.truncated" class="mt-3 text-sm text-muted-foreground">Showing the first 200 services.</p>
        <p v-if="page.groups.length === 0" class="mt-8 text-center text-muted-foreground">No services on this page.</p>

        <section
          v-for="(group, groupIndex) in page.groups"
          :key="group.name || '__without-group__'"
          class="mt-8"
          :aria-labelledby="`status-group-${groupIndex}`"
          :data-testid="`status-group-${group.name || 'outros'}`"
        >
          <div class="flex items-center justify-between gap-4 border-b pb-2 dark:border-gray-800">
            <h2 :id="`status-group-${groupIndex}`" class="text-lg font-semibold">{{ group.name || 'Other services' }}</h2>
            <span :class="['text-sm', groupStatusClass(group.status)]">{{ groupStatusLabel(group.status) }}</span>
          </div>
          <ul class="divide-y dark:divide-gray-800">
            <EndpointRow v-for="endpoint in group.endpoints" :key="endpoint.name" :endpoint="endpoint" :bars="bars" />
          </ul>
        </section>
      </template>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import Loading from '@/components/Loading.vue'
import StatusSummary from '@/components/public/StatusSummary.vue'
import EndpointRow from '@/components/public/EndpointRow.vue'
import { SLUG_PATTERN, STATUS_LABELS } from '@/utils/statusPage'

const REFRESH_INTERVAL_MS = 60000
const CLOCK_INTERVAL_MS = 10000

const route = useRoute()

const page = ref(null)
// loading, ready (with or without page, see errorMessage) or not-found
const state = ref('loading')
const errorMessage = ref('')
const now = ref(Date.now())
const narrowScreen = window.matchMedia('(max-width: 639px)')
const bars = ref(narrowScreen.matches ? 25 : 50)

let refreshTimer = null
let clockTimer = null
let abortController = null
let requestGeneration = 0

const slug = computed(() => (route.name === 'PublicStatusPage' ? String(route.params.slug || '') : ''))

const groupStatusLabel = (status) => STATUS_LABELS[status]?.group || STATUS_LABELS.unknown.group

const groupStatusClass = (status) => ({
  operational: 'text-green-700 dark:text-green-400',
  degraded: 'text-amber-700 dark:text-amber-400',
  down: 'text-red-700 dark:text-red-400'
}[status] || 'text-muted-foreground')

const stopRefreshing = () => {
  clearTimeout(refreshTimer)
  refreshTimer = null
}

const scheduleRefresh = (delayMs) => {
  stopRefreshing()
  if (document.visibilityState !== 'hidden') {
    refreshTimer = setTimeout(load, delayMs)
  }
}

const showNotFound = () => {
  stopRefreshing()
  page.value = null
  errorMessage.value = ''
  state.value = 'not-found'
  document.title = 'Page not found'
}

const load = async () => {
  const currentSlug = slug.value
  const generation = ++requestGeneration
  abortController?.abort()
  if (!SLUG_PATTERN.test(currentSlug)) {
    showNotFound()
    return
  }
  abortController = new AbortController()
  let retryDelayMs = REFRESH_INTERVAL_MS
  try {
    const response = await fetch(`/api/v1/status-pages/${encodeURIComponent(currentSlug)}`, { credentials: 'omit', signal: abortController.signal })
    if (generation !== requestGeneration) {
      return
    }
    if (response.status === 404) {
      showNotFound()
      return
    }
    if (response.status === 429 || response.status === 503) {
      const retryAfterSeconds = Number(response.headers.get('Retry-After'))
      if (retryAfterSeconds > REFRESH_INTERVAL_MS / 1000) {
        retryDelayMs = retryAfterSeconds * 1000
      }
      errorMessage.value = response.status === 429
        ? 'Too many requests right now. This page will refresh automatically.'
        : 'This status page is temporarily unavailable. It will refresh automatically.'
    } else if (!response.ok || !(response.headers.get('Content-Type') || '').includes('application/json')) {
      errorMessage.value = 'Could not load the status page. It will refresh automatically.'
    } else {
      page.value = await response.json()
      errorMessage.value = ''
      document.title = page.value.title
    }
  } catch (error) {
    if (error.name === 'AbortError' || generation !== requestGeneration) {
      return
    }
    errorMessage.value = 'Could not reach the server. This page will refresh automatically.'
  }
  now.value = Date.now()
  state.value = 'ready'
  scheduleRefresh(retryDelayMs)
}

const handleVisibilityChange = () => {
  if (document.visibilityState === 'hidden') {
    stopRefreshing()
  } else if (state.value !== 'not-found') {
    load()
  }
}

const handleScreenChange = (event) => {
  bars.value = event.matches ? 25 : 50
}

watch(slug, () => {
  page.value = null
  errorMessage.value = ''
  state.value = 'loading'
  load()
})

onMounted(() => {
  load()
  clockTimer = setInterval(() => { now.value = Date.now() }, CLOCK_INTERVAL_MS)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  narrowScreen.addEventListener('change', handleScreenChange)
})

onUnmounted(() => {
  stopRefreshing()
  clearInterval(clockTimer)
  abortController?.abort()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  narrowScreen.removeEventListener('change', handleScreenChange)
})
</script>
