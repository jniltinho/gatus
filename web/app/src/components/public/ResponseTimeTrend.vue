<template>
  <div class="mt-3 border p-3 dark:border-gray-800" :data-testid="`status-chart-${endpoint.name}`">
    <div class="mb-2 flex items-center justify-between gap-2">
      <p class="text-sm font-medium">Response time trend</p>
      <select
        v-model="duration"
        class="border border-input bg-background px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-ring dark:border-gray-700"
        :aria-label="`Period of the response time chart of ${endpoint.name}`"
        :data-testid="`status-chart-duration-${endpoint.name}`"
      >
        <option v-for="option in RESPONSE_TIME_DURATIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
    </div>
    <div class="relative w-full" style="height: 220px;">
      <div v-if="loading" class="absolute inset-0 flex items-center justify-center"><Loading /></div>
      <p v-else-if="error" class="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">{{ error }}</p>
      <p v-else-if="points.length === 0" class="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">No response time data for this period.</p>
      <Line v-else :data="chartData" :options="chartOptions" role="img" :aria-label="`Average response time of ${endpoint.name} per hour`" />
    </div>
  </div>
</template>

<script setup>
// Response time chart of an endpoint of a public status page, in the format of ResponseTimeChart (endpoint details page
// of the dashboard), with the points of the public response-times route instead of the key of the endpoint
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, Filler, LinearScale, LineElement, PointElement, TimeScale, Tooltip } from 'chart.js'
import 'chartjs-adapter-date-fns'
import Loading from '@/components/Loading.vue'
import { RESPONSE_TIME_DURATIONS } from '@/utils/statusPage'

ChartJS.register(LinearScale, PointElement, LineElement, Tooltip, Filler, TimeScale)

const props = defineProps({
  endpoint: { type: Object, required: true },
  group: { type: String, default: '' }
})

const loadResponseTimes = inject('loadResponseTimes')

const duration = ref('24h')
const loading = ref(true)
const error = ref('')
const points = ref([])
const isDark = ref(document.documentElement.classList.contains('dark'))

let requestId = 0
let observer = null

const load = async () => {
  const id = ++requestId
  loading.value = true
  error.value = ''
  try {
    const data = await loadResponseTimes(duration.value)
    if (id !== requestId) {
      return
    }
    const group = props.group.trim()
    const series = (data.endpoints || []).find((item) => item.name === props.endpoint.name && (item.group || '').trim() === group)
    points.value = series ? series.points : []
  } catch (e) {
    if (id !== requestId) {
      return
    }
    points.value = []
    error.value = 'Could not load the response time chart.'
  } finally {
    if (id === requestId) {
      loading.value = false
    }
  }
}

const chartData = computed(() => ({
  labels: points.value.map((point) => new Date(point.timestamp)),
  datasets: [{
    label: 'Response Time (ms)',
    data: points.value.map((point) => point.ms),
    borderColor: isDark.value ? 'rgb(96, 165, 250)' : 'rgb(59, 130, 246)',
    backgroundColor: isDark.value ? 'rgba(96, 165, 250, 0.1)' : 'rgba(59, 130, 246, 0.1)',
    borderWidth: 2,
    pointRadius: 2,
    pointHoverRadius: 4,
    tension: 0.1,
    fill: true
  }]
}))

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? false : undefined,
  interaction: { mode: 'index', intersect: false },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: isDark.value ? 'rgba(31, 41, 55, 0.95)' : 'rgba(255, 255, 255, 0.95)',
      titleColor: isDark.value ? '#f9fafb' : '#111827',
      bodyColor: isDark.value ? '#d1d5db' : '#374151',
      borderColor: isDark.value ? '#4b5563' : '#e5e7eb',
      borderWidth: 1,
      padding: 12,
      cornerRadius: 0,
      displayColors: false,
      callbacks: {
        title: (items) => (items.length > 0 ? new Date(items[0].parsed.x).toLocaleString() : ''),
        label: (context) => `${context.parsed.y}ms`
      }
    }
  },
  scales: {
    x: {
      type: 'time',
      time: {
        unit: duration.value === '24h' ? 'hour' : 'day',
        displayFormats: { hour: 'MMM d, ha', day: 'MMM d' }
      },
      grid: { color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)', drawBorder: false },
      ticks: { color: isDark.value ? '#9ca3af' : '#6b7280', maxRotation: 0, autoSkipPadding: 20 }
    },
    y: {
      beginAtZero: true,
      grid: { color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)', drawBorder: false },
      ticks: { color: isDark.value ? '#9ca3af' : '#6b7280', callback: (value) => `${value}ms` }
    }
  }
}))

watch(duration, load)

onMounted(() => {
  load()
  observer = new MutationObserver(() => {
    isDark.value = document.documentElement.classList.contains('dark')
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onUnmounted(() => {
  requestId++
  if (observer) {
    observer.disconnect()
  }
})
</script>
