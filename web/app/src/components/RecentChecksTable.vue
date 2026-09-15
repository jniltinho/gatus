<template>
  <div class="overflow-x-auto border dark:border-gray-700" data-testid="recent-checks-table">
    <table class="w-full text-sm">
      <thead class="bg-muted/50 text-left text-muted-foreground dark:bg-gray-800 dark:text-gray-400">
        <tr>
          <th class="px-3 py-2 font-medium">Status</th>
          <th class="px-3 py-2 font-medium">Date and time</th>
          <th class="px-3 py-2 font-medium">Origin</th>
          <th class="px-3 py-2 font-medium">Message</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="rows.length === 0">
          <td colspan="4" class="px-3 py-6 text-center text-muted-foreground dark:text-gray-400">No checks yet.</td>
        </tr>
        <tr v-for="(row, index) in rows" :key="`${row.timestamp}-${index}`" class="border-t dark:border-gray-700" :data-testid="`recent-check-${index}`">
          <td class="px-3 py-2">
            <span :class="['border px-1.5 py-0.5 text-xs font-medium', row.success ? 'border-green-300 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-300' : 'border-red-300 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-300']">{{ row.success ? 'Up' : 'Down' }}</span>
          </td>
          <td class="px-3 py-2 whitespace-nowrap text-muted-foreground dark:text-gray-400">{{ row.dateTime }}</td>
          <td class="px-3 py-2 whitespace-nowrap">
            <span :class="['text-xs', row.push ? 'text-violet-700 dark:text-violet-300' : 'text-muted-foreground dark:text-gray-400']">{{ row.push ? 'Push' : 'Check' }}</span>
          </td>
          <td class="px-3 py-2 break-words text-foreground dark:text-gray-200" data-testid="recent-check-message">{{ row.message }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
// Table of the results of the current page, from the most recent to the oldest, with the message of each one (fork)
import { computed } from 'vue'

const props = defineProps({
  results: { type: Array, default: () => [] },
})

// The message of a push; otherwise the errors; otherwise the HTTP status of the check
const messageOf = (result) => {
  if (result.message) {
    return result.message
  }
  if (result.errors && result.errors.length) {
    return result.errors.join('; ')
  }
  return result.status ? `HTTP ${result.status}` : ''
}

const rows = computed(() => [...props.results].reverse().map((result) => ({
  timestamp: result.timestamp,
  success: result.success,
  dateTime: new Date(result.timestamp).toLocaleString(),
  push: result.origin === 'push',
  message: messageOf(result),
})))
</script>
