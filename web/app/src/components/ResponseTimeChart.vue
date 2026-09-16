<template>
  <div class="relative w-full" style="height: 300px;">
    <div v-if="loading" class="absolute inset-0 flex items-center justify-center bg-background/50">
      <Loading />
    </div>
    <div v-else-if="error" class="absolute inset-0 flex items-center justify-center text-muted-foreground">
      {{ error }}
    </div>
    <Line v-else :data="chartData" :options="chartOptions" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler, TimeScale } from 'chart.js'
import annotationPlugin from 'chartjs-plugin-annotation'
import 'chartjs-adapter-date-fns'
import { generatePrettyTimeDifference } from '@/utils/time'
import { downtimeIntervals } from '@/utils/downtime'
import Loading from './Loading.vue'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler, TimeScale, annotationPlugin)

const props = defineProps({
  endpointKey: {
    type: String,
    required: true
  },
  duration: {
    type: String,
    required: true,
    validator: (value) => ['24h', '7d', '30d'].includes(value)
  },
  serverUrl: {
    type: String,
    default: '..'
  },
  events: {
    type: Array,
    default: () => []
  }
})

const loading = ref(true)
const error = ref(null)
const timestamps = ref([])
const values = ref([])
const isDark = ref(document.documentElement.classList.contains('dark'))
const hoveredEventIndex = ref(null)

const DURATIONS_MS = {
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
  '30d': 30 * 24 * 60 * 60 * 1000
}

// End of the period of the chart, updated with the data so that the period follows the refreshes (fork)
const periodEnd = ref(Date.now())
const periodStart = computed(() => periodEnd.value - DURATIONS_MS[props.duration])

// Fork: periods out of service, from each UNHEALTHY event to the next HEALTHY event, clipped to the period of the chart
const downtimes = computed(() => {
  const firstPoint = timestamps.value.length > 0 ? new Date(timestamps.value[0]).getTime() : null
  return downtimeIntervals(props.events, periodStart.value, periodEnd.value, firstPoint).map((interval) => ({
    ...interval,
    ongoing: interval.end === periodEnd.value,
    duration: generatePrettyTimeDifference(interval.end, interval.start)
  }))
})

const chartData = computed(() => {
  if (timestamps.value.length === 0) {
    return {
      labels: [],
      datasets: []
    }
  }
  const labels = timestamps.value.map(ts => new Date(ts))
  return {
    labels,
    datasets: [{
      label: 'Response Time (ms)',
      data: values.value,
      borderColor: isDark.value ? 'rgb(96, 165, 250)' : 'rgb(59, 130, 246)',
      backgroundColor: isDark.value ? 'rgba(96, 165, 250, 0.1)' : 'rgba(59, 130, 246, 0.1)',
      borderWidth: 2,
      pointRadius: 2,
      pointHoverRadius: 4,
      tension: 0.1,
      fill: true
    }]
  }
})

const chartOptions = computed(() => {
  // Include hoveredEventIndex in dependency tracking
  // eslint-disable-next-line no-unused-vars
  const _ = hoveredEventIndex.value

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false
    },
    plugins: {
      legend: {
        display: false
      },
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
          title: (tooltipItems) => {
            if (tooltipItems.length > 0) {
              const date = new Date(tooltipItems[0].parsed.x)
              return date.toLocaleString()
            }
            return ''
          },
          label: (context) => {
            const value = context.parsed.y
            return `${value}ms`
          }
        }
      },
      annotation: {
        // Fork: translucent red boxes over the periods out of service, in the place of the dashed lines of the events
        annotations: downtimes.value.reduce((acc, downtime, index) => {
          acc[`downtime-${index}`] = {
            type: 'box',
            xMin: downtime.start,
            xMax: downtime.end,
            backgroundColor: isDark.value ? 'rgba(248, 113, 113, 0.2)' : 'rgba(239, 68, 68, 0.15)',
            borderColor: isDark.value ? 'rgba(248, 113, 113, 0.5)' : 'rgba(239, 68, 68, 0.4)',
            borderWidth: 1,
            enter() {
              hoveredEventIndex.value = index
            },
            leave() {
              hoveredEventIndex.value = null
            },
            label: {
              borderRadius: 0,
              display: () => hoveredEventIndex.value === index,
              content: [downtime.ongoing ? 'Status: ONGOING' : 'Status: RESOLVED', `Down for ${downtime.duration}`, `Started at ${new Date(downtime.start).toLocaleString()}`],
              backgroundColor: 'rgba(239, 68, 68, 0.9)',
              color: '#ffffff',
              font: {
                size: 11
              },
              padding: 6,
              position: { x: 'center', y: 'start' }
            }
          }
          return acc
        }, {})
      }
    },
    scales: {
      x: {
        type: 'time',
        // Fork: the axis is fixed to the period, so that the periods out of service do not stretch it
        min: periodStart.value,
        max: periodEnd.value,
        time: {
          unit: props.duration === '24h' ? 'hour' : props.duration === '7d' ? 'day' : 'day',
          displayFormats: {
            hour: 'MMM d, ha',
            day: 'MMM d'
          }
        },
        grid: {
          color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)',
          drawBorder: false
        },
        ticks: {
          color: isDark.value ? '#9ca3af' : '#6b7280',
          maxRotation: 0,
          autoSkipPadding: 20
        }
      },
      y: {
        beginAtZero: true,
        grid: {
          color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)',
          drawBorder: false
        },
        ticks: {
          color: isDark.value ? '#9ca3af' : '#6b7280',
          callback: (value) => `${value}ms`
        }
      }
    }
  }
})

const fetchData = async () => {
  loading.value = true
  error.value = null
  periodEnd.value = Date.now()
  try {
    const response = await fetch(`${props.serverUrl}/api/v1/endpoints/${props.endpointKey}/response-times/${props.duration}/history`, {
      credentials: 'include'
    })
    if (response.status === 200) {
      const data = await response.json()
      timestamps.value = data.timestamps || []
      values.value = data.values || []
    } else {
      error.value = 'Failed to load chart data'
      console.error('[ResponseTimeChart] Error:', await response.text())
    }
  } catch (err) {
    error.value = 'Failed to load chart data'
    console.error('[ResponseTimeChart] Error:', err)
  } finally {
    loading.value = false
  }
}

watch(() => props.duration, () => {
  fetchData()
})

// Fork: the events come with the refreshes of the page, so an ongoing period out of service follows the current time
watch(() => props.events, () => {
  periodEnd.value = Date.now()
})

onMounted(() => {
  fetchData()
  const observer = new MutationObserver(() => {
    isDark.value = document.documentElement.classList.contains('dark')
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  onUnmounted(() => observer.disconnect())
})
</script>