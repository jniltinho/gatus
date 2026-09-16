<template>
  <div class="container mx-auto px-4 py-8 max-w-5xl">
    <RouterLink
      v-if="validAddress"
      :to="{ name: 'PublicStatusPage', params: { slug } }"
      class="mb-4 inline-flex h-9 items-center gap-2 px-3 text-sm font-medium hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:hover:bg-gray-800"
      data-testid="status-endpoint-back"
    >
      <ArrowLeft class="h-4 w-4" aria-hidden="true" />
      Back to {{ details ? details.page.title : 'the status page' }}
    </RouterLink>

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
        :class="['mb-4 border px-4 py-3 text-sm', details ? 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100' : 'border-red-300 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200']"
      >
        {{ errorMessage }}
      </div>

      <div v-if="details" class="space-y-6" data-testid="status-endpoint-details">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div class="min-w-0">
            <h1 class="text-4xl font-bold tracking-tight break-words" data-testid="status-endpoint-name">{{ details.name }}</h1>
            <p class="mt-2 text-muted-foreground">
              <span v-if="details.group">Group: {{ details.group }} · </span>Updated {{ relativeTimeLabel(details.updatedAt, now) }}
            </p>
            <!-- Fork: expiration of the TLS certificate, when the page shows it -->
            <p v-if="certificateDays !== null" :class="['mt-1 text-xs', certificateClass(certificateDays)]" data-testid="status-endpoint-certificate">
              {{ certificateText(certificateDays) }}
            </p>
          </div>
          <StatusBadge :status="healthStatus" />
        </div>

        <!-- Fork: same order as the monitor page of the Uptime Kuma and the endpoint details page of the dashboard -->
        <Card data-testid="status-endpoint-recent-checks">
          <CardHeader>
            <CardTitle>Recent Checks</CardTitle>
          </CardHeader>
          <CardContent>
            <ul>
              <EndpointRow :endpoint="details" :group="details.group" :bars="bars" :show-header="false" />
            </ul>
          </CardContent>
        </Card>

        <!-- Fork: same panel of numbers as the dashboard, with the uptimes and the averages of the payload -->
        <DetailsSummary
          :current-response-time="lastResult ? lastResult.durationMs : null"
          :uptime="details.uptime"
          :response-time="details.responseTime"
        />

        <Card v-if="hasResponseTimes" data-testid="status-endpoint-chart">
          <CardHeader>
            <div class="flex items-center justify-between gap-4">
              <CardTitle>Response Time Trend</CardTitle>
              <select
                v-model="chartDuration"
                aria-label="Period of the response time chart"
                class="border border-input bg-background px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-ring dark:border-gray-700"
                data-testid="status-endpoint-chart-duration"
              >
                <option v-for="option in RESPONSE_TIME_DURATIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </div>
          </CardHeader>
          <CardContent>
            <ResponseTimeChart :key="key" :endpoint-key="key" :duration="chartDuration" server-url="" :events="details.events" />
          </CardContent>
        </Card>

        <Card v-if="results.length > 0" data-testid="status-endpoint-checks-table">
          <div class="p-6">
            <RecentChecksTable :results="results" :show-message="showMessages" sanitized />
          </div>
        </Card>

        <div v-if="hasResponseTimes" class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card v-for="period in BADGE_PERIODS" :key="period.value">
            <CardHeader class="pb-2">
              <CardTitle class="text-sm font-medium text-muted-foreground text-center">{{ period.label }}</CardTitle>
            </CardHeader>
            <CardContent>
              <img :src="badgeURL(`response-times/${period.value}/badge.svg`)" :alt="`Average response time over the ${period.label.toLowerCase()}`" class="mx-auto mt-2" />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Current Health</CardTitle>
          </CardHeader>
          <CardContent>
            <div class="text-center">
              <img :src="badgeURL('health/badge.svg')" alt="Current health" class="mx-auto" />
            </div>
          </CardContent>
        </Card>

        <Card v-if="events.length > 0" data-testid="status-endpoint-events">
          <CardHeader>
            <CardTitle>Events</CardTitle>
          </CardHeader>
          <CardContent>
            <ul class="space-y-4">
              <li v-for="event in events" :key="`${event.type}-${event.timestamp}`" class="flex items-start gap-4 pb-4 border-b last:border-0 dark:border-gray-800">
                <div class="mt-1" aria-hidden="true">
                  <ArrowUpCircle v-if="event.type === 'HEALTHY'" class="h-5 w-5 text-green-500" />
                  <ArrowDownCircle v-else-if="event.type === 'UNHEALTHY'" class="h-5 w-5 text-red-500" />
                  <PlayCircle v-else class="h-5 w-5 text-muted-foreground" />
                </div>
                <div class="flex-1">
                  <p class="font-medium">{{ event.text }}</p>
                  <p class="text-sm text-muted-foreground">{{ formatDateTime(event.timestamp) }} • {{ event.timeAgo }}</p>
                </div>
              </li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { ArrowDownCircle, ArrowLeft, ArrowUpCircle, PlayCircle } from 'lucide-vue-next'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import Loading from '@/components/Loading.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import ResponseTimeChart from '@/components/ResponseTimeChart.vue'
import EndpointRow from '@/components/public/EndpointRow.vue'
import RecentChecksTable from '@/components/RecentChecksTable.vue'
import DetailsSummary from '@/components/DetailsSummary.vue'
import { describeEvents, formatDateTime, relativeTimeLabel, RESPONSE_TIME_DURATIONS, SLUG_PATTERN } from '@/utils/statusPage'
import { certificateClass, certificateText } from '@/utils/certificate'

const REFRESH_INTERVAL_MS = 60000
const CLOCK_INTERVAL_MS = 10000
const MAXIMUM_KEY_LENGTH = 400

// Periods of the badges, like the endpoint details page of the dashboard
const BADGE_PERIODS = [
  { value: '30d', label: 'Last 30 days' },
  { value: '7d', label: 'Last 7 days' },
  { value: '24h', label: 'Last 24 hours' },
  { value: '1h', label: 'Last hour' }
]

const route = useRoute()

const details = ref(null)
// loading, ready (with or without details, see errorMessage) or not-found
const state = ref('loading')
const errorMessage = ref('')
const now = ref(Date.now())
const chartDuration = ref('24h')
const narrowScreen = window.matchMedia('(max-width: 639px)')
const bars = ref(narrowScreen.matches ? 25 : 50)

let refreshTimer = null
let clockTimer = null
let abortController = null
let requestGeneration = 0

const slug = computed(() => String(route.params.slug || ''))
const key = computed(() => String(route.params.key || ''))
const validAddress = computed(() => SLUG_PATTERN.test(slug.value) && key.value.length > 0 && key.value.length <= MAXIMUM_KEY_LENGTH)

const results = computed(() => (details.value && details.value.results) || [])
const lastResult = computed(() => (results.value.length > 0 ? results.value[results.value.length - 1] : null))
const events = computed(() => (details.value ? describeEvents(details.value.events || []) : []))
// Days until the TLS certificate expires, only published when the page shows it (fork)
const certificateDays = computed(() => (details.value && Number.isInteger(details.value.certificateExpiresInDays) ? details.value.certificateExpiresInDays : null))
// Like the dashboard, which shows the chart as soon as a result has a duration: results faster than 1 ms have a
// durationMs of 0 in the public payload, but still have points in the chart
const hasResponseTimes = computed(() => results.value.length > 0)

const healthStatus = computed(() => ({ up: 'healthy', pending: 'pending', down: 'unhealthy' }[details.value?.status] || 'unknown'))
// With show-messages, the table of checks has the same columns as the dashboard, even without any message (fork)
const showMessages = computed(() => details.value?.page?.showMessages === true)

// badgeURL returns the address of a badge of the endpoint, public in the original Gatus
const badgeURL = (path) => `/api/v1/endpoints/${encodeURIComponent(key.value)}/${path}`

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
  details.value = null
  errorMessage.value = ''
  state.value = 'not-found'
  document.title = 'Page not found'
}

const load = async () => {
  const generation = ++requestGeneration
  abortController?.abort()
  if (!validAddress.value) {
    showNotFound()
    return
  }
  abortController = new AbortController()
  let retryDelayMs = REFRESH_INTERVAL_MS
  try {
    const response = await fetch(`/api/v1/status-pages/${encodeURIComponent(slug.value)}/endpoints/${encodeURIComponent(key.value)}`, { credentials: 'omit', signal: abortController.signal })
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
        : 'This page is temporarily unavailable. It will refresh automatically.'
    } else if (!response.ok || !(response.headers.get('Content-Type') || '').includes('application/json')) {
      errorMessage.value = 'Could not load the details of the service. It will refresh automatically.'
    } else {
      details.value = await response.json()
      errorMessage.value = ''
      document.title = `${details.value.name} · ${details.value.page.title}`
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

watch([slug, key], () => {
  details.value = null
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
