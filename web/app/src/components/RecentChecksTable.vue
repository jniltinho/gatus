<template>
  <div>
    <button
      type="button"
      :aria-expanded="expanded"
      data-testid="recent-checks-toggle"
      class="flex w-full items-center gap-2 border px-3 py-2 text-left text-sm font-medium text-foreground hover:bg-accent dark:border-gray-700 dark:text-gray-200 dark:hover:bg-gray-800"
      @click="toggle"
    >
      <ChevronDown v-if="expanded" class="h-4 w-4" />
      <ChevronRight v-else class="h-4 w-4" />
      Checks table
      <span class="font-normal text-muted-foreground dark:text-gray-400">({{ rows.length }})</span>
      <span class="ml-auto text-xs font-normal text-muted-foreground dark:text-gray-400">{{ showMessage ? 'Status, date and time, message and origin' : 'Status, date and time and response time' }}</span>
    </button>
    <div v-if="expanded" class="overflow-x-auto border border-t-0 dark:border-gray-700" data-testid="recent-checks-table">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-muted-foreground dark:bg-gray-800 dark:text-gray-400">
          <tr>
            <th class="px-3 py-2 font-medium">Status</th>
            <th class="px-3 py-2 font-medium">Date and time</th>
            <template v-if="showMessage">
              <th class="px-3 py-2 font-medium">Message</th>
              <th class="px-3 py-2 font-medium">Origin</th>
            </template>
            <th v-else class="px-3 py-2 font-medium">Response time</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="rows.length === 0">
            <td :colspan="showMessage ? 4 : 3" class="px-3 py-6 text-center text-muted-foreground dark:text-gray-400">No checks yet.</td>
          </tr>
          <tr v-for="(row, index) in rows" :key="`${row.timestamp}-${index}`" class="border-t dark:border-gray-700" :data-testid="`recent-check-${index}`">
            <td class="px-3 py-2">
              <span :class="['border px-1.5 py-0.5 text-xs font-medium', row.success ? 'border-green-300 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-300' : 'border-red-300 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-300']">{{ row.success ? 'Up' : 'Down' }}</span>
            </td>
            <td class="px-3 py-2 whitespace-nowrap text-muted-foreground dark:text-gray-400">{{ row.dateTime }}</td>
            <template v-if="showMessage">
              <td class="px-3 py-2 break-words text-foreground dark:text-gray-200" data-testid="recent-check-message">{{ row.message }}</td>
              <td class="px-3 py-2 whitespace-nowrap">
                <span :class="['text-xs', row.push ? 'text-violet-700 dark:text-violet-300' : 'text-muted-foreground dark:text-gray-400']">{{ row.push ? 'Push' : 'Check' }}</span>
              </td>
            </template>
            <td v-else class="px-3 py-2 whitespace-nowrap text-muted-foreground dark:text-gray-400">{{ row.responseTime }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
// Table of the results of the current page, from the most recent to the oldest (fork). The columns follow the table of
// the Uptime Kuma (Status, DateTime, Message), with the origin at the end. On the public status pages (showMessage
// false), the results have no message nor errors, so the table shows the response time instead. It starts collapsed,
// and the choice of the viewer is remembered in the browser.
import { computed, ref } from 'vue'
import { ChevronDown, ChevronRight } from 'lucide-vue-next'

const props = defineProps({
  results: { type: Array, default: () => [] },
  showMessage: { type: Boolean, default: true },
})

const STORAGE_KEY = 'gatus:show-recent-checks-table'

const readExpanded = () => {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch (e) {
    return false
  }
}

const expanded = ref(readExpanded())

const toggle = () => {
  expanded.value = !expanded.value
  try {
    localStorage.setItem(STORAGE_KEY, expanded.value ? 'true' : 'false')
  } catch (e) {
    // The table keeps working without the browser storage
  }
}

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

// Public results have durationMs; the results of the dashboard have duration in nanoseconds
const responseTimeOf = (result) => {
  const milliseconds = result.durationMs !== undefined ? result.durationMs : Math.trunc((result.duration || 0) / 1000000)
  return milliseconds > 0 ? `${milliseconds}ms` : '—'
}

const rows = computed(() => [...props.results].reverse().map((result) => ({
  timestamp: result.timestamp,
  success: result.success,
  dateTime: new Date(result.timestamp).toLocaleString(),
  push: result.origin === 'push',
  message: props.showMessage ? messageOf(result) : '',
  responseTime: responseTimeOf(result),
})))
</script>
