<template>
  <li class="py-3" :data-testid="`status-endpoint-${endpoint.name}`">
    <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-center gap-2 min-w-0">
        <span :class="['inline-block h-2.5 w-2.5 rounded-full flex-shrink-0', dotClass]" aria-hidden="true"></span>
        <span class="font-medium truncate" :title="endpoint.name">{{ endpoint.name }}</span>
        <span :class="['text-xs', statusTextClass]" aria-hidden="true">{{ statusLabel }}</span>
      </div>
      <dl class="flex gap-4 text-xs text-muted-foreground" aria-hidden="true">
        <div v-for="period in periods" :key="period.key" class="flex gap-1">
          <dt>{{ period.label }}</dt>
          <dd class="font-medium text-foreground">{{ formatUptime(endpoint.uptime[period.key]) }}</dd>
        </div>
      </dl>
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
        {{ formatDateTime(activeResult.timestamp) }} · {{ activeResult.success ? 'Success' : 'Failure' }} · {{ activeResult.durationMs }} ms
      </template>
    </p>
  </li>
</template>

<script setup>
import { computed, ref } from 'vue'
import { formatDateTime, formatUptime, STATUS_LABELS } from '@/utils/statusPage'

const props = defineProps({
  endpoint: { type: Object, required: true },
  bars: { type: Number, default: 50 }
})

const periods = [
  { key: '24h', label: '24h' },
  { key: '7d', label: '7d' },
  { key: '30d', label: '30d' }
]

const hoveredIndex = ref(null)
const selectedIndex = ref(null)

const displayedResults = computed(() => {
  const results = (props.endpoint.results || []).slice(-props.bars)
  return [...Array(props.bars - results.length).fill(null), ...results]
})

const activeIndex = computed(() => (hoveredIndex.value !== null ? hoveredIndex.value : selectedIndex.value))
const activeResult = computed(() => (activeIndex.value !== null ? displayedResults.value[activeIndex.value] : null))

const statusLabel = computed(() => STATUS_LABELS[props.endpoint.status]?.endpoint || STATUS_LABELS.unknown.endpoint)

const dotClass = computed(() => ({ up: 'bg-green-500', down: 'bg-red-500' }[props.endpoint.status] || 'bg-gray-400'))

const statusTextClass = computed(() => ({
  up: 'text-green-700 dark:text-green-400',
  down: 'text-red-700 dark:text-red-400'
}[props.endpoint.status] || 'text-muted-foreground'))

const accessibleSummary = computed(() => {
  const results = props.endpoint.results || []
  const successes = results.filter((result) => result.success).length
  return `${props.endpoint.name}: ${statusLabel.value}, ${successes} of ${results.length} checks successful, 24-hour uptime ${formatUptime(props.endpoint.uptime['24h'])}`
})

const barClass = (result, index) => {
  if (!result) {
    return 'bg-gray-200 dark:bg-gray-800'
  }
  const active = activeIndex.value === index
  if (result.success) {
    return active ? 'bg-green-700' : 'bg-green-500'
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
