<template>
  <div class="container mx-auto px-4 py-8 max-w-7xl">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground dark:text-gray-100">Push keys</h1>
        <p class="text-muted-foreground dark:text-gray-400 mt-1">Global keys accept push for every endpoint that receives push, at /api/push/&lt;key&gt;/&lt;endpoint-key&gt;</p>
      </div>
      <div class="flex gap-2">
        <router-link to="/" class="inline-flex h-10 items-center border border-input bg-background px-4 text-sm font-medium hover:bg-accent dark:border-gray-700 dark:hover:bg-gray-800">Dashboard</router-link>
      </div>
    </div>

    <AdminTabs active="push-keys" />

    <div v-if="listing && listing.managedUnavailable" role="alert" data-testid="push-keys-unavailable" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">
      The push keys created through the web could not be loaded: only the ones from the configuration file are accepted until the next reload.
    </div>
    <div v-if="notice" role="status" data-testid="admin-notice" class="mb-4 border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200">{{ notice }}</div>
    <div v-if="error" role="alert" data-testid="admin-error" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">{{ error }}</div>

    <div v-if="created" role="status" data-testid="push-key-created" class="mb-4 border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100">
      <p class="font-medium">Copy the key {{ created.name }} now: it will not be shown again.</p>
      <div class="mt-2 flex gap-2">
        <input :value="created.token" readonly class="h-10 w-full border border-input bg-background px-3 font-mono text-sm text-foreground dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100" data-testid="push-key-token" @focus="$event.target.select()" />
        <Button variant="outline" size="sm" data-testid="push-key-copy" @click="copyText(created.token)">Copy</Button>
      </div>
      <p class="mt-2">Example: <span class="break-all font-mono">{{ exampleUrl(created.token) }}</span></p>
      <Button variant="ghost" size="sm" class="mt-2" data-testid="push-key-done" @click="created = null">Done</Button>
    </div>

    <form class="mb-4 flex flex-wrap items-end gap-2 border bg-card p-4 dark:border-gray-700 dark:bg-gray-900" @submit.prevent="create">
      <label class="block min-w-0 flex-1 text-sm font-medium text-foreground dark:text-gray-200">Name of the new key
        <Input v-model="name" placeholder="e.g. akamai" maxlength="64" class="mt-1 dark:border-gray-700" data-testid="push-key-name" />
      </label>
      <Button type="submit" :disabled="busy || !name.trim()" data-testid="push-key-create">Create key</Button>
    </form>

    <div v-if="loading" class="py-12 flex justify-center"><Loading /></div>
    <div v-else class="overflow-x-auto border bg-card dark:border-gray-700 dark:bg-gray-900">
      <table class="w-full text-sm" data-testid="push-keys-table">
        <thead class="bg-muted/50 text-left text-muted-foreground dark:bg-gray-800 dark:text-gray-400">
          <tr>
            <th class="px-3 py-2 font-medium">Name</th>
            <th class="px-3 py-2 font-medium">Key</th>
            <th class="px-3 py-2 font-medium">Source</th>
            <th class="px-3 py-2 font-medium">Created</th>
            <th class="px-3 py-2 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="keys.length === 0">
            <td colspan="5" class="px-3 py-8 text-center text-muted-foreground dark:text-gray-400">No push keys yet.</td>
          </tr>
          <tr v-for="key in keys" :key="`${key.origin}-${key.id || key.name}`" class="border-t dark:border-gray-700" :data-testid="`push-key-row-${key.origin}-${key.name}`">
            <td class="px-3 py-2 font-medium text-foreground dark:text-gray-100">{{ key.name }}</td>
            <td class="px-3 py-2 font-mono text-xs text-muted-foreground dark:text-gray-400">{{ key.hint ? `…${key.hint}` : '—' }}</td>
            <td class="px-3 py-2">
              <span :class="['border px-1.5 py-0.5 text-xs', key.origin === 'admin' ? 'border-blue-300 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-200' : 'border-gray-300 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300']">{{ key.origin === 'admin' ? 'Web' : 'YAML' }}</span>
            </td>
            <td class="px-3 py-2 text-muted-foreground dark:text-gray-400">
              <template v-if="key.createdAt">{{ new Date(key.createdAt).toLocaleString() }}<span v-if="key.createdBy"> · {{ key.createdBy }}</span></template>
              <template v-else>—</template>
            </td>
            <td class="px-3 py-2 whitespace-nowrap text-right">
              <Button v-if="key.origin === 'admin'" variant="ghost" size="sm" class="text-red-600 dark:text-red-400" :disabled="busy" :data-testid="`push-key-revoke-${key.name}`" @click="pendingRevocation = key">Revoke</Button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmDialog
      :open="pendingRevocation !== null"
      title="Revoke push key"
      :message="revocationMessage"
      confirm-label="Revoke"
      @confirm="confirmRevocation"
      @cancel="pendingRevocation = null"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Loading from '@/components/Loading.vue'
import ConfirmDialog from '@/components/admin/ConfirmDialog.vue'
import AdminTabs from '@/components/admin/AdminTabs.vue'
import { describePushKeyError, pushKeysApi } from '@/utils/adminApi'

const listing = ref(null)
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const name = ref('')
const created = ref(null)
const pendingRevocation = ref(null)

const keys = computed(() => (listing.value && listing.value.keys) || [])

const revocationMessage = computed(() => (pendingRevocation.value ? `Push key ${pendingRevocation.value.name} will be revoked: pushes with it are rejected immediately.` : ''))

const exampleUrl = (token) => `${window.location.origin}/api/push/${encodeURIComponent(token)}/<endpoint-key>?status=up&msg=OK&ping=`

const load = async () => {
  try {
    const { data } = await pushKeysApi.list()
    listing.value = data
  } catch (e) {
    error.value = describePushKeyError(e)
  } finally {
    loading.value = false
  }
}

const create = async () => {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const { data } = await pushKeysApi.create(name.value.trim())
    created.value = data
    name.value = ''
  } catch (e) {
    error.value = describePushKeyError(e)
  } finally {
    busy.value = false
    await load()
  }
}

const copyText = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    notice.value = 'Key copied to the clipboard.'
  } catch (e) {
    notice.value = 'Select the key and copy it.'
  }
}

const confirmRevocation = async () => {
  const key = pendingRevocation.value
  pendingRevocation.value = null
  if (!key) {
    return
  }
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await pushKeysApi.remove(key.id)
    notice.value = `Push key ${key.name} revoked.`
  } catch (e) {
    error.value = describePushKeyError(e)
  } finally {
    busy.value = false
    await load()
  }
}

onMounted(load)
</script>
