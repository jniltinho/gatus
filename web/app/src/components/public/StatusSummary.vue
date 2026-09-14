<template>
  <div :class="['border px-4 py-3 flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between', bannerClass]" data-testid="status-summary">
    <p role="status" class="font-semibold flex items-center gap-2">
      <span :class="['inline-block h-3 w-3 rounded-full flex-shrink-0', dotClass]" aria-hidden="true"></span>
      {{ label }}
    </p>
    <p class="text-sm opacity-80" data-testid="status-summary-updated">{{ updatedLabel }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { relativeTimeLabel, STATUS_LABELS } from '@/utils/statusPage'

const props = defineProps({
  status: { type: String, required: true },
  updatedAt: { type: String, required: true },
  now: { type: Number, required: true }
})

const label = computed(() => STATUS_LABELS[props.status]?.page || STATUS_LABELS.unknown.page)

// The Tailwind version of the project has no 950 shade: dark backgrounds use the 900 shade with opacity
const bannerClass = computed(() => ({
  operational: 'border-green-300 bg-green-50 text-green-900 dark:border-green-800 dark:bg-green-900/30 dark:text-green-100',
  degraded: 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100',
  down: 'border-red-300 bg-red-50 text-red-900 dark:border-red-800 dark:bg-red-900/30 dark:text-red-100'
}[props.status] || 'border-gray-300 bg-gray-50 text-gray-800 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200'))

const dotClass = computed(() => ({
  operational: 'bg-green-500',
  degraded: 'bg-amber-500',
  down: 'bg-red-500'
}[props.status] || 'bg-gray-400'))

const updatedLabel = computed(() => `Updated ${relativeTimeLabel(props.updatedAt, props.now)}`)
</script>
