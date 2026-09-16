<template>
  <li :class="featured ? 'border bg-card p-4 dark:border-gray-800' : 'py-3'" :data-testid="`status-endpoint-${endpoint.name}`">
    <div v-if="showHeader" class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-center gap-2 min-w-0">
        <span :class="['inline-block h-2.5 w-2.5 rounded-full flex-shrink-0', dotClass]" aria-hidden="true"></span>
        <component
          :is="slug ? RouterLink : 'span'"
          :to="slug ? detailsRoute : undefined"
          :class="['truncate', featured ? 'text-lg font-semibold' : 'font-medium', slug ? 'underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring' : '']"
          :title="endpoint.name"
          :data-testid="slug ? `status-endpoint-link-${endpoint.name}` : undefined"
        >{{ endpoint.name }}</component>
        <span :class="['text-xs', statusTextClass]" aria-hidden="true">{{ statusLabel }}</span>
        <span v-if="featured && group" class="truncate text-xs text-muted-foreground" :title="group">{{ group }}</span>
      </div>
      <dl v-if="!featured" class="flex gap-4 text-xs text-muted-foreground" aria-hidden="true">
        <div v-for="period in periods" :key="period.key" class="flex gap-1">
          <dt>{{ period.label }}</dt>
          <dd class="font-medium text-foreground">{{ formatUptime(endpoint.uptime[period.key]) }}</dd>
        </div>
      </dl>
    </div>
    <!-- Fork: expiration of the TLS certificate, when the page shows it -->
    <p v-if="showHeader && certificateDays !== null" :class="['mt-0.5 text-xs', certificateClass(certificateDays)]" :data-testid="`status-endpoint-certificate-${endpoint.name}`">
      {{ certificateText(certificateDays) }}
    </p>
    <table v-if="featured" class="mt-3 w-full text-sm" data-testid="status-featured-stats">
      <thead>
        <tr class="text-xs text-muted-foreground">
          <th scope="col" class="py-1 pr-2 text-left font-normal"><span class="sr-only">Metric</span></th>
          <th v-for="period in periods" :key="`period-${period.key}`" scope="col" class="py-1 pl-2 text-right font-normal">{{ period.label }}</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <th scope="row" class="py-1 pr-2 text-left text-xs font-normal text-muted-foreground">Uptime</th>
          <td v-for="period in periods" :key="`uptime-${period.key}`" class="py-1 pl-2 text-right font-medium">{{ formatUptime(endpoint.uptime[period.key]) }}</td>
        </tr>
        <tr>
          <th scope="row" class="py-1 pr-2 text-left text-xs font-normal text-muted-foreground">Avg response</th>
          <td v-for="period in periods" :key="`response-time-${period.key}`" class="py-1 pl-2 text-right font-medium">{{ formatMilliseconds(responseTime[period.key]) }}</td>
        </tr>
      </tbody>
    </table>
    <div v-if="featured" class="mt-1 flex items-center justify-between gap-4 text-xs text-muted-foreground">
      <p>Last response <span class="font-medium text-foreground">{{ formatMilliseconds(lastResult ? lastResult.durationMs : null) }}</span></p>
      <RouterLink
        v-if="slug"
        :to="detailsRoute"
        class="font-medium text-foreground underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        :data-testid="`status-endpoint-details-${endpoint.name}`"
      >View details<span class="sr-only"> of {{ endpoint.name }}</span></RouterLink>
    </div>
    <p class="sr-only">{{ accessibleSummary }}</p>
    <div
      class="mt-2 flex gap-px outline-none focus-visible:ring-2 focus-visible:ring-ring"
      role="group"
      tabindex="0"
      :aria-label="`Check history of ${endpoint.name}. Use the arrow keys to browse it.`"
      @keydown="handleKeydown"
      @mouseleave="hoveredIndex = null"
      @blur="selectedIndex = null"
    >
      <span
        v-for="(result, index) in displayedResults"
        :key="index"
        aria-hidden="true"
        :class="['h-6 flex-1', barClass(result, index)]"
        @mouseenter="result && (hoveredIndex = index)"
        @click="result && selectBar(index)"
      ></span>
    </div>
    <p class="mt-1 min-h-[1rem] text-xs text-muted-foreground" aria-live="polite" data-testid="status-endpoint-detail">
      <template v-if="activeResult">
        {{ formatDateTime(activeResult.timestamp) }} · {{ activeResult.success ? 'Success' : (activeResult.pending ? 'Pending' : 'Failure') }} · {{ activeResult.durationMs }} ms
      </template>
    </p>
  </li>
</template>

<script setup>
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { endpointKey, formatDateTime, formatMilliseconds, formatUptime, STATUS_LABELS } from '@/utils/statusPage'
import { certificateClass, certificateText } from '@/utils/certificate'

const props = defineProps({
  endpoint: { type: Object, required: true },
  bars: { type: Number, default: 50 },
  // Featured endpoints are shown as a card, with more details
  featured: { type: Boolean, default: false },
  group: { type: String, default: '' },
  // Slug of the status page: when set, the name of the endpoint links to its details page
  slug: { type: String, default: '' },
  // The details page shows the bars without the name and the uptimes, which it already shows
  showHeader: { type: Boolean, default: true }
})

const periods = [
  { key: '24h', label: '24h' },
  { key: '7d', label: '7d' },
  { key: '30d', label: '30d' }
]

const hoveredIndex = ref(null)
const selectedIndex = ref(null)

const detailsRoute = computed(() => ({
  name: 'PublicStatusPageEndpoint',
  params: { slug: props.slug, key: endpointKey(props.group, props.endpoint.name) }
}))

const displayedResults = computed(() => {
  const results = (props.endpoint.results || []).slice(-props.bars)
  return [...Array(props.bars - results.length).fill(null), ...results]
})

const lastResult = computed(() => {
  const results = props.endpoint.results || []
  return results.length > 0 ? results[results.length - 1] : null
})

const responseTime = computed(() => props.endpoint.responseTime || {})

// Days until the TLS certificate expires, only published when the page shows it (fork)
const certificateDays = computed(() => (Number.isInteger(props.endpoint.certificateExpiresInDays) ? props.endpoint.certificateExpiresInDays : null))

const activeIndex = computed(() => (hoveredIndex.value !== null ? hoveredIndex.value : selectedIndex.value))
const activeResult = computed(() => (activeIndex.value !== null ? displayedResults.value[activeIndex.value] : null))

const statusLabel = computed(() => STATUS_LABELS[props.endpoint.status]?.endpoint || STATUS_LABELS.unknown.endpoint)

const dotClass = computed(() => ({ up: 'bg-green-500', pending: 'bg-yellow-400', down: 'bg-red-500' }[props.endpoint.status] || 'bg-gray-400'))

const statusTextClass = computed(() => ({
  up: 'text-green-700 dark:text-green-400',
  pending: 'text-yellow-700 dark:text-yellow-400',
  down: 'text-red-700 dark:text-red-400'
}[props.endpoint.status] || 'text-muted-foreground'))

const accessibleSummary = computed(() => {
  const results = props.endpoint.results || []
  const successes = results.filter((result) => result.success).length
  return `${props.endpoint.name}: ${statusLabel.value}, ${successes} of ${results.length} checks successful, 24-hour uptime ${formatUptime(props.endpoint.uptime['24h'])}, 24-hour average response time ${formatMilliseconds(responseTime.value['24h'])}`
})

const barClass = (result, index) => {
  if (!result) {
    return 'bg-gray-200 dark:bg-gray-800'
  }
  const active = activeIndex.value === index
  if (result.success) {
    return active ? 'bg-green-700' : 'bg-green-500'
  }
  // Fork: Pending results are not successful, but are shown in yellow
  if (result.pending) {
    return active ? 'bg-yellow-600' : 'bg-yellow-400'
  }
  return active ? 'bg-red-700' : 'bg-red-500'
}

const firstResultIndex = () => displayedResults.value.findIndex((result) => result !== null)

const selectBar = (index) => {
  selectedIndex.value = selectedIndex.value === index ? null : index
}

const handleKeydown = (event) => {
  const first = firstResultIndex()
  if (first < 0) {
    return
  }
  const last = displayedResults.value.length - 1
  const current = selectedIndex.value === null ? last + 1 : selectedIndex.value
  const moves = {
    ArrowLeft: Math.max(first, current - 1),
    ArrowRight: Math.min(last, current + 1),
    Home: first,
    End: last
  }
  if (event.key === 'Escape') {
    selectedIndex.value = null
    return
  }
  if (event.key in moves) {
    event.preventDefault()
    selectedIndex.value = moves[event.key]
  }
}
</script>
