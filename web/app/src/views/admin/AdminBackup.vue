<template>
  <!-- Backup and restore of the items managed through the web (fork) -->
  <div class="container mx-auto max-w-5xl px-4 py-4" data-testid="admin-backup">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="min-w-0">
        <h1 class="text-xl font-semibold tracking-tight text-foreground dark:text-gray-100">Backup</h1>
        <p class="mt-0.5 text-sm text-muted-foreground dark:text-gray-400">Endpoints, status pages and push keys created through the web. History and the configuration file are not included.</p>
      </div>
      <div class="flex shrink-0 flex-wrap items-center gap-2">
        <router-link to="/" class="inline-flex h-9 items-center border border-input bg-background px-3 text-sm font-medium hover:bg-accent dark:border-gray-700 dark:hover:bg-gray-800">Dashboard</router-link>
      </div>
    </div>

    <AdminTabs active="backup" class="mt-3" />

    <div class="space-y-4">
      <!-- Download -->
      <section class="border bg-card p-5 dark:border-gray-700 dark:bg-gray-900" data-testid="backup-section-download">
        <header class="mb-4">
          <h2 class="text-base font-semibold text-foreground dark:text-gray-100">Download backup</h2>
          <p class="mt-0.5 text-xs text-muted-foreground dark:text-gray-400">A JSON file with the complete definitions, including tokens, passwords, headers and webhooks, that can be restored in another installation with any supported database.</p>
        </header>

        <div class="grid gap-2 sm:grid-cols-3">
          <div v-for="counter in counters" :key="counter.id" class="border px-3 py-2 dark:border-gray-700">
            <span class="block text-xs text-muted-foreground dark:text-gray-400">{{ counter.label }}</span>
            <span class="block text-lg font-semibold text-foreground dark:text-gray-100" :data-testid="`backup-count-${counter.id}`">{{ counter.value === null ? '—' : counter.value }}</span>
          </div>
        </div>
        <p v-if="countsError" role="alert" class="mt-2 text-xs text-red-700 dark:text-red-300" data-testid="backup-counts-error">{{ countsError }}</p>

        <div class="mt-4 grid gap-4 sm:grid-cols-2">
          <label :class="['flex items-start gap-3 border px-3 py-2.5 text-sm sm:col-span-2 dark:border-gray-700', encrypt ? 'border-green-300 bg-green-50/60 dark:border-green-800 dark:bg-green-900/20' : '']">
            <input v-model="encrypt" type="checkbox" class="mt-0.5 h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="backup-encrypt" />
            <span>
              <span class="block font-medium text-foreground dark:text-gray-200">Encrypt with a password</span>
              <span class="block text-xs text-muted-foreground dark:text-gray-400">Argon2id and AES-256-GCM. The password is not stored anywhere: without it the file cannot be restored.</span>
            </span>
          </label>

          <template v-if="encrypt">
            <label class="block">
              <span class="flex items-baseline justify-between text-sm font-medium text-foreground dark:text-gray-200">
                Password
                <span class="text-xs font-normal text-muted-foreground dark:text-gray-400" data-testid="backup-password-bytes">{{ downloadPasswordBytes }}/{{ MAX_PASSWORD_BYTES }} bytes</span>
              </span>
              <Input v-model="downloadPassword" type="password" autocomplete="new-password" class="mt-1.5 dark:border-gray-700" data-testid="backup-password" />
            </label>
            <label class="block">
              <span class="block text-sm font-medium text-foreground dark:text-gray-200">Confirm the password</span>
              <Input v-model="downloadPasswordConfirmation" type="password" autocomplete="new-password" class="mt-1.5 dark:border-gray-700" data-testid="backup-password-confirm" />
            </label>
            <p class="text-xs text-muted-foreground sm:col-span-2 dark:text-gray-400" data-testid="backup-password-hint">
              <span v-if="downloadPasswordError && (downloadPassword || downloadPasswordConfirmation)" class="text-red-700 dark:text-red-300">{{ downloadPasswordError }}</span>
              <span v-else>Between {{ MIN_PASSWORD_BYTES }} and {{ MAX_PASSWORD_BYTES }} bytes in UTF-8.</span>
            </p>
          </template>

          <div v-else role="status" class="border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 sm:col-span-2 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100" data-testid="backup-plaintext-warning">
            Without a password, the file contains tokens, passwords, headers and webhooks in plain text. Keep it somewhere safe.
          </div>
        </div>

        <div v-if="downloadError" role="alert" data-testid="backup-error" class="mt-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">{{ downloadError }}</div>
        <div v-if="downloadNotice" role="status" data-testid="backup-notice" class="mt-4 border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200">{{ downloadNotice }}</div>

        <div class="mt-4 flex flex-wrap items-center gap-2 border-t pt-4 dark:border-gray-700">
          <Button :disabled="downloading || (encrypt && Boolean(downloadPasswordError))" data-testid="backup-download" @click="download">{{ downloading ? 'Downloading…' : 'Download' }}</Button>
        </div>
      </section>

      <!-- Restore -->
      <section class="border bg-card p-5 dark:border-gray-700 dark:bg-gray-900" data-testid="backup-section-restore">
        <header class="mb-4">
          <h2 class="text-base font-semibold text-foreground dark:text-gray-100">Restore</h2>
          <p class="mt-0.5 text-xs text-muted-foreground dark:text-gray-400">Merges a backup into this installation: nothing is removed, and the preview shows exactly what will be applied.</p>
        </header>

        <div class="grid gap-4 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label for="restore-file" class="block text-sm font-medium text-foreground dark:text-gray-200">Backup file</label>
            <div class="mt-1.5 flex gap-2">
              <input
                id="restore-file"
                ref="fileInput"
                type="file"
                class="flex h-10 w-full min-w-0 border border-input bg-background px-3 py-2 text-sm text-foreground file:mr-3 file:border-0 file:bg-transparent file:text-sm file:font-medium dark:border-gray-700 dark:text-gray-100"
                data-testid="restore-file"
                @change="onFileChange"
              />
              <Button variant="outline" class="shrink-0" :disabled="!selectedFile || reading || previewing || restoring" data-testid="restore-reload" @click="readFile(selectedFile)">Read again</Button>
            </div>
            <p class="mt-1 text-xs text-muted-foreground dark:text-gray-400" data-testid="restore-file-status">
              <template v-if="reading">Reading the file…</template>
              <template v-else-if="fileFormat === 'plain'">Backup in plain text.</template>
              <template v-else-if="fileFormat === 'encrypted'">Encrypted backup: enter its password.</template>
              <template v-else>Any file up to 2.7 MiB; the format is detected from its content.</template>
            </p>
          </div>

          <label v-if="fileFormat === 'encrypted'" class="block sm:col-span-2">
            <span class="block text-sm font-medium text-foreground dark:text-gray-200">Password of the backup</span>
            <Input v-model="restorePassword" type="password" autocomplete="off" class="mt-1.5 dark:border-gray-700" data-testid="restore-password" />
            <span v-if="restorePassword && restorePasswordError" class="mt-1 block text-xs text-red-700 dark:text-red-300" data-testid="restore-password-hint">{{ restorePasswordError }}</span>
          </label>

          <label class="flex items-start gap-3 border px-3 py-2.5 text-sm dark:border-gray-700">
            <input v-model="overwrite" type="checkbox" class="mt-0.5 h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="restore-overwrite" />
            <span>
              <span class="block font-medium text-foreground dark:text-gray-200">Overwrite existing items</span>
              <span class="block text-xs text-muted-foreground dark:text-gray-400">Items that already exist with a different definition are updated. Without it they are skipped.</span>
            </span>
          </label>
          <label class="flex items-start gap-3 border px-3 py-2.5 text-sm dark:border-gray-700">
            <input v-model="disableEndpoints" type="checkbox" class="mt-0.5 h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="restore-disable-endpoints" />
            <span>
              <span class="block font-medium text-foreground dark:text-gray-200">Restore endpoints as disabled</span>
              <span class="block text-xs text-muted-foreground dark:text-gray-400">Created or updated endpoints are saved disabled: no checks and no alerts until they are enabled.</span>
            </span>
          </label>
        </div>

        <div v-if="restoreError" role="alert" data-testid="restore-error" class="mt-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">{{ restoreError }}</div>

        <div class="mt-4 flex flex-wrap items-center gap-2 border-t pt-4 dark:border-gray-700">
          <Button variant="outline" :disabled="!canPreview" data-testid="restore-preview" @click="runPreview">{{ previewing ? 'Previewing…' : 'Preview' }}</Button>
          <Button variant="destructive" class="sm:ml-auto" :disabled="!canRestore" data-testid="restore-apply" @click="confirming = true">{{ restoring ? 'Restoring…' : 'Restore' }}</Button>
        </div>

        <!-- Plan -->
        <div v-if="plan" class="mt-4 space-y-3" data-testid="restore-plan">
          <div v-if="notices.monitoringStarts || notices.withAlerts" role="status" class="border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100" data-testid="restore-notices">
            <p v-if="notices.monitoringStarts" data-testid="restore-notice-monitoring">
              {{ notices.monitoringStarts }} enabled {{ notices.monitoringStarts === 1 ? 'endpoint' : 'endpoints' }} will start monitoring right away, from the network of this installation.
            </p>
            <p v-if="notices.withAlerts" data-testid="restore-notice-alerts">
              {{ notices.withAlerts }} of them {{ notices.withAlerts === 1 ? 'has' : 'have' }} alerts that can be sent to the real providers.
            </p>
          </div>

          <div class="flex flex-wrap items-center justify-between gap-2">
            <p class="text-sm text-foreground dark:text-gray-100" data-testid="restore-summary">{{ planSummaryText(plan) }}</p>
            <div class="flex flex-wrap" role="group" aria-label="Filter by action">
              <button
                v-for="option in filterOptions"
                :key="option.id"
                type="button"
                :aria-pressed="actionFilter === option.id"
                :class="[
                  '-ml-px h-9 border px-3 text-xs font-medium first:ml-0 dark:border-gray-700',
                  actionFilter === option.id ? 'relative z-10 border-gray-900 bg-gray-900 text-white dark:border-gray-100 dark:bg-gray-100 dark:text-gray-900' : 'bg-background text-foreground hover:bg-accent dark:text-gray-200 dark:hover:bg-gray-800'
                ]"
                :data-testid="`restore-filter-${option.id}`"
                @click="actionFilter = option.id"
              >{{ option.label }} ({{ option.count }})</button>
            </div>
          </div>

          <div class="overflow-x-auto border dark:border-gray-700">
            <table class="w-full text-sm" data-testid="restore-plan-table">
              <thead class="bg-gray-50 text-left text-muted-foreground dark:bg-gray-800 dark:text-gray-400">
                <tr>
                  <th class="px-3 py-2 font-medium">Type</th>
                  <th class="px-3 py-2 font-medium">ID</th>
                  <th class="px-3 py-2 font-medium">Action</th>
                  <th class="px-3 py-2 font-medium">Reason and warnings</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="filteredItems.length === 0">
                  <td colspan="4" class="px-3 py-6 text-center text-muted-foreground dark:text-gray-400">No items.</td>
                </tr>
                <tr
                  v-for="item in filteredItems"
                  :key="`${item.type}-${item.id}`"
                  class="border-t align-top dark:border-gray-700"
                  :data-testid="`restore-plan-row-${item.type}-${item.id}`"
                  :data-action="item.action"
                >
                  <td class="whitespace-nowrap px-3 py-1.5 text-muted-foreground dark:text-gray-400">{{ typeLabel(item.type) }}</td>
                  <td class="px-3 py-1.5 font-mono text-xs text-foreground break-all dark:text-gray-100">{{ item.id }}</td>
                  <td class="px-3 py-1.5"><span :class="['border px-1.5 py-0.5 text-xs', badgeClass(item.action)]">{{ item.action }}</span></td>
                  <td class="px-3 py-1.5 text-xs">
                    <span v-if="item.reason" class="text-foreground dark:text-gray-200">{{ item.reason }}</span>
                    <ul v-if="item.warnings && item.warnings.length" class="list-disc pl-4 text-amber-800 dark:text-amber-300">
                      <li v-for="(warning, index) in item.warnings" :key="index">{{ formatWarning(warning) }}</li>
                    </ul>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Results -->
        <div v-if="results" class="mt-4 space-y-3" data-testid="restore-results">
          <div role="status" class="border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200" data-testid="restore-results-summary">
            Restore finished: {{ resultSummary.created }} created · {{ resultSummary.updated }} updated · {{ resultSummary.unchanged }} unchanged · {{ resultSummary.skipped }} skipped · {{ resultSummary.failed }} failed.
          </div>
          <div class="overflow-x-auto border dark:border-gray-700">
            <table class="w-full text-sm" data-testid="restore-results-table">
              <thead class="bg-gray-50 text-left text-muted-foreground dark:bg-gray-800 dark:text-gray-400">
                <tr>
                  <th class="px-3 py-2 font-medium">Type</th>
                  <th class="px-3 py-2 font-medium">ID</th>
                  <th class="px-3 py-2 font-medium">Result</th>
                  <th class="px-3 py-2 font-medium">Message and warnings</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="results.length === 0">
                  <td colspan="4" class="px-3 py-6 text-center text-muted-foreground dark:text-gray-400">No items.</td>
                </tr>
                <tr
                  v-for="item in results"
                  :key="`${item.type}-${item.id}`"
                  class="border-t align-top dark:border-gray-700"
                  :data-testid="`restore-result-row-${item.type}-${item.id}`"
                  :data-result="item.result"
                >
                  <td class="whitespace-nowrap px-3 py-1.5 text-muted-foreground dark:text-gray-400">{{ typeLabel(item.type) }}</td>
                  <td class="px-3 py-1.5 font-mono text-xs text-foreground break-all dark:text-gray-100">{{ item.id }}</td>
                  <td class="px-3 py-1.5"><span :class="['border px-1.5 py-0.5 text-xs', badgeClass(item.result)]">{{ item.result }}</span></td>
                  <td class="px-3 py-1.5 text-xs">
                    <span v-if="item.message" class="text-foreground dark:text-gray-200">{{ item.message }}</span>
                    <ul v-if="item.warnings && item.warnings.length" class="list-disc pl-4 text-amber-800 dark:text-amber-300">
                      <li v-for="(warning, index) in item.warnings" :key="index">{{ formatWarning(warning) }}</li>
                    </ul>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </div>

    <ConfirmDialog
      :open="confirming"
      title="Restore backup"
      :message="confirmationMessage"
      confirm-label="Restore"
      @confirm="applyRestore"
      @cancel="confirming = false"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import AdminTabs from '@/components/admin/AdminTabs.vue'
import ConfirmDialog from '@/components/admin/ConfirmDialog.vue'
import { adminApi, backupApi, pushKeysApi, statusPagesApi } from '@/utils/adminApi'
import {
  MAX_PASSWORD_BYTES,
  MIN_PASSWORD_BYTES,
  buildRestoreBody,
  describeBackupError,
  detectBackupFormat,
  filterPlanItems,
  formatWarning,
  isRestoreFileTooLarge,
  passwordByteLength,
  planCounts,
  planSummaryText,
  restoreConfirmationMessage,
  resultCounts,
  validatePassword
} from '@/utils/adminBackup'

// Quantities of the items managed through the web
const counts = ref({ endpoints: null, statusPages: null, pushKeys: null })
const countsError = ref('')

const counters = computed(() => [
  { id: 'endpoints', label: 'Managed endpoints', value: counts.value.endpoints },
  { id: 'status-pages', label: 'Managed status pages', value: counts.value.statusPages },
  { id: 'push-keys', label: 'Web push keys', value: counts.value.pushKeys }
])

const loadCounts = async () => {
  const [endpoints, statusPages, pushKeys] = await Promise.allSettled([adminApi.list(), statusPagesApi.list(), pushKeysApi.list()])
  const failures = []
  const value = (outcome, pick) => {
    if (outcome.status === 'fulfilled') {
      return pick(outcome.value.data)
    }
    failures.push(outcome.reason)
    return null
  }
  counts.value = {
    endpoints: value(endpoints, (data) => (Array.isArray(data) ? data : []).filter((item) => item.source === 'admin').length),
    statusPages: value(statusPages, (data) => ((data && data.statusPages) || []).filter((item) => item.origin === 'admin').length),
    pushKeys: value(pushKeys, (data) => ((data && data.keys) || []).filter((item) => item.origin === 'admin').length)
  }
  countsError.value = failures.length ? `The quantities could not be loaded: ${describeBackupError(failures[0])}` : ''
}

// Download
const encrypt = ref(false)
const downloadPassword = ref('')
const downloadPasswordConfirmation = ref('')
const downloading = ref(false)
const downloadError = ref('')
const downloadNotice = ref('')

const downloadPasswordBytes = computed(() => passwordByteLength(downloadPassword.value))
const downloadPasswordError = computed(() => validatePassword(downloadPassword.value, downloadPasswordConfirmation.value))

const saveBlob = (blob, filename) => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

const download = async () => {
  if (encrypt.value && downloadPasswordError.value) {
    downloadError.value = downloadPasswordError.value
    return
  }
  downloading.value = true
  downloadError.value = ''
  downloadNotice.value = ''
  try {
    const { blob, filename } = await backupApi.download(encrypt.value ? downloadPassword.value : '')
    saveBlob(blob, filename)
    downloadNotice.value = `Backup downloaded as ${filename}.`
  } catch (e) {
    downloadError.value = describeBackupError(e)
  } finally {
    downloading.value = false
  }
}

// Restore
const fileInput = ref(null)
const selectedFile = ref(null)
const parsedFile = ref(null)
const fileFormat = ref('')
const reading = ref(false)
const restorePassword = ref('')
const overwrite = ref(false)
const disableEndpoints = ref(false)
const previewing = ref(false)
const restoring = ref(false)
const confirming = ref(false)
const restoreError = ref('')
const plan = ref(null)
const results = ref(null)
const actionFilter = ref('all')
// Each change of the file, of the password or of the options starts a new generation: answers of older ones are ignored
let generation = 0

const invalidate = () => {
  generation++
  plan.value = null
  results.value = null
  confirming.value = false
  actionFilter.value = 'all'
}

watch([restorePassword, overwrite, disableEndpoints], () => {
  invalidate()
  restoreError.value = ''
})

const readFile = async (file) => {
  invalidate()
  restoreError.value = ''
  parsedFile.value = null
  fileFormat.value = ''
  if (!file) {
    return
  }
  if (isRestoreFileTooLarge(file.size)) {
    restoreError.value = 'The file is too large: a backup has at most 2 MiB, about 2.7 MiB when encrypted.'
    return
  }
  const current = generation
  reading.value = true
  try {
    const text = await file.text()
    if (current !== generation) {
      return
    }
    let parsed
    try {
      parsed = JSON.parse(text)
    } catch (e) {
      restoreError.value = 'The file is not a backup: it is not valid JSON.'
      return
    }
    const format = detectBackupFormat(parsed)
    if (format === 'unknown') {
      restoreError.value = 'The file is not a backup: unknown format.'
      return
    }
    parsedFile.value = parsed
    fileFormat.value = format
  } catch (e) {
    if (current === generation) {
      restoreError.value = 'The file could not be read.'
    }
  } finally {
    if (current === generation) {
      reading.value = false
    }
  }
}

const onFileChange = (event) => {
  const file = (event.target.files && event.target.files[0]) || null
  selectedFile.value = file
  reading.value = false
  readFile(file)
}

const restorePasswordError = computed(() => (fileFormat.value === 'encrypted' ? validatePassword(restorePassword.value) : ''))

const canPreview = computed(() => Boolean(parsedFile.value) && !reading.value && !previewing.value && !restoring.value && !restorePasswordError.value)
const canRestore = computed(() => Boolean(plan.value && plan.value.fingerprint) && !reading.value && !previewing.value && !restoring.value)

const restoreBody = (fingerprint) => buildRestoreBody({
  file: parsedFile.value,
  format: fileFormat.value,
  password: restorePassword.value,
  overwrite: overwrite.value,
  disableEndpoints: disableEndpoints.value,
  fingerprint
})

const runPreview = async () => {
  if (!canPreview.value) {
    return
  }
  invalidate()
  const current = generation
  previewing.value = true
  restoreError.value = ''
  try {
    const { data } = await backupApi.preview(restoreBody())
    if (current === generation) {
      plan.value = data
    }
  } catch (e) {
    if (current === generation) {
      restoreError.value = describeBackupError(e)
    }
  } finally {
    previewing.value = false
  }
}

const applyRestore = async () => {
  confirming.value = false
  if (!canRestore.value) {
    return
  }
  restoring.value = true
  restoreError.value = ''
  try {
    const { data } = await backupApi.restore(restoreBody(plan.value.fingerprint))
    // The plan was applied, even if the options changed meanwhile: a new restore needs a new preview
    invalidate()
    results.value = (data && data.results) || []
    loadCounts()
  } catch (e) {
    restoreError.value = describeBackupError(e)
    if (e && e.status === 409) {
      invalidate()
    }
  } finally {
    restoring.value = false
  }
}

const notices = computed(() => (plan.value && plan.value.notices) || { monitoringStarts: 0, withAlerts: 0 })

const filterOptions = computed(() => {
  const counted = planCounts(plan.value)
  const items = (plan.value && plan.value.items) || []
  return [
    { id: 'all', label: 'All', count: items.length },
    { id: 'create', label: 'Create', count: counted.create },
    { id: 'update', label: 'Update', count: counted.update },
    { id: 'unchanged', label: 'Unchanged', count: counted.unchanged },
    { id: 'skip', label: 'Skip', count: counted.skip }
  ]
})

const filteredItems = computed(() => filterPlanItems(plan.value && plan.value.items, actionFilter.value))

const confirmationMessage = computed(() => `${restoreConfirmationMessage(plan.value)}\nNothing is removed, and items that are skipped in the preview are not changed.`)

const resultSummary = computed(() => resultCounts(results.value))

const TYPE_LABELS = { pushKey: 'Push key', endpoint: 'Endpoint', statusPage: 'Status page' }
const typeLabel = (type) => TYPE_LABELS[type] || type

const BADGES = {
  green: 'border-green-300 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200',
  blue: 'border-blue-300 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-200',
  gray: 'border-gray-300 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300',
  amber: 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100',
  red: 'border-red-300 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200'
}
const BADGE_COLORS = { create: 'green', created: 'green', update: 'blue', updated: 'blue', unchanged: 'gray', skip: 'amber', skipped: 'amber', failed: 'red' }
const badgeClass = (value) => BADGES[BADGE_COLORS[value] || 'gray']

onMounted(loadCounts)
</script>
