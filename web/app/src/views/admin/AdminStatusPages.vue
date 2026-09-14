<template>
  <div class="container mx-auto px-4 py-8 max-w-7xl">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground dark:text-gray-100">Status pages</h1>
        <p class="text-muted-foreground dark:text-gray-400 mt-1">Public pages, open without login at /status/&lt;slug&gt;</p>
      </div>
      <div class="flex gap-2">
        <router-link to="/" class="inline-flex h-10 items-center border border-input bg-background px-4 text-sm font-medium hover:bg-accent dark:border-gray-700 dark:hover:bg-gray-800">Dashboard</router-link>
        <Button data-testid="admin-new-status-page" @click="router.push({ name: 'AdminStatusPageNew' })">New status page</Button>
      </div>
    </div>

    <AdminTabs active="status-pages" />

    <div v-if="listing && !listing.publicationEnabled" role="status" data-testid="status-pages-disabled" class="mb-4 border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100">
      Status pages are disabled in the configuration file (<code>status-pages.enabled: false</code>): no page is published, but you can still edit and preview them.
    </div>
    <div v-if="listing && listing.managedUnavailable" role="alert" data-testid="status-pages-unavailable" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">
      The status pages managed through the web could not be loaded: only the ones from the configuration file are published until the next reload.
    </div>
    <div v-if="listing && listing.sharedRateLimitWarning" role="status" data-testid="status-pages-shared-limit" class="mb-4 border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100">
      Visits come from <code>{{ listing.sharedRateLimitWarning }}</code> with X-Forwarded-For, but this IP is not in <code>status-pages.trusted-proxies</code>: all visitors share the same rate limit.
    </div>

    <div v-if="notice" role="status" data-testid="admin-notice" class="mb-4 border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200">{{ notice }}</div>
    <div v-if="error" role="alert" data-testid="admin-error" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">{{ error }}</div>
    <div v-if="copyFallbackUrl" class="mb-4 border bg-card px-4 py-3 text-sm dark:border-gray-700 dark:bg-gray-900">
      <label class="block text-foreground dark:text-gray-200">Copy the address of the page:
        <input ref="copyInput" :value="copyFallbackUrl" readonly class="mt-1 w-full border border-input bg-background px-2 py-1 font-mono text-sm dark:border-gray-700" data-testid="status-page-copy-fallback" @focus="$event.target.select()" />
      </label>
    </div>

    <div v-if="loading" class="py-12 flex justify-center"><Loading /></div>
    <div v-else class="overflow-x-auto border bg-card dark:border-gray-700 dark:bg-gray-900">
      <table class="w-full text-sm" data-testid="status-pages-table">
        <thead class="bg-muted/50 text-left text-muted-foreground dark:bg-gray-800 dark:text-gray-400">
          <tr>
            <th class="px-3 py-2 font-medium">Slug</th>
            <th class="px-3 py-2 font-medium">Title</th>
            <th class="px-3 py-2 font-medium">Source</th>
            <th class="px-3 py-2 font-medium">Status</th>
            <th class="px-3 py-2 font-medium">Endpoints</th>
            <th class="px-3 py-2 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="items.length === 0">
            <td colspan="6" class="px-3 py-8 text-center text-muted-foreground dark:text-gray-400">No status pages yet.</td>
          </tr>
          <tr v-for="item in items" :key="`${item.origin}-${item.slug}`" class="border-t dark:border-gray-700" :data-testid="`status-page-row-${item.origin}-${item.slug}`">
            <td class="px-3 py-2 font-mono text-xs text-foreground dark:text-gray-100">{{ item.slug }}</td>
            <td class="px-3 py-2 font-medium text-foreground dark:text-gray-100">{{ item.title || '—' }}</td>
            <td class="px-3 py-2">
              <span :class="['border px-1.5 py-0.5 text-xs', item.origin === 'admin' ? 'border-blue-300 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-200' : 'border-gray-300 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300']">{{ item.origin === 'admin' ? 'Web' : 'YAML' }}</span>
            </td>
            <td class="px-3 py-2">
              <span :class="stateClass(item)" :title="item.conflictOrigin || item.error || ''">{{ stateLabel(item) }}</span>
            </td>
            <td class="px-3 py-2">{{ item.endpoints }}</td>
            <td class="px-3 py-2 whitespace-nowrap text-right">
              <a :href="item.path" target="_blank" rel="noopener" class="inline-flex h-9 items-center px-3 text-sm font-medium hover:bg-accent dark:hover:bg-gray-800" :data-testid="`status-page-open-${item.slug}`">Open</a>
              <Button variant="ghost" size="sm" :data-testid="`status-page-copy-${item.slug}`" @click="copyLink(item)">Copy link</Button>
              <Button variant="ghost" size="sm" :data-testid="`status-page-edit-${item.slug}`" @click="edit(item)">{{ item.origin === 'admin' ? 'Edit' : 'View' }}</Button>
              <template v-if="item.origin === 'admin'">
                <Button variant="ghost" size="sm" :disabled="busySlug === item.slug || item.conflict || Boolean(item.error)" :data-testid="`status-page-toggle-${item.slug}`" @click="toggle(item)">{{ item.enabled ? 'Disable' : 'Enable' }}</Button>
                <Button variant="ghost" size="sm" class="text-red-600 dark:text-red-400" :disabled="busySlug === item.slug" :data-testid="`status-page-remove-${item.slug}`" @click="pendingRemoval = item">Remove</Button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmDialog
      :open="pendingRemoval !== null"
      title="Remove status page"
      :message="removalMessage"
      confirm-label="Remove"
      @confirm="confirmRemoval"
      @cancel="pendingRemoval = null"
    />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import Loading from '@/components/Loading.vue'
import ConfirmDialog from '@/components/admin/ConfirmDialog.vue'
import AdminTabs from '@/components/admin/AdminTabs.vue'
import { describeStatusPageError, statusPagesApi } from '@/utils/adminApi'

const router = useRouter()

const listing = ref(null)
const loading = ref(true)
const error = ref('')
const notice = ref('')
const busySlug = ref('')
const pendingRemoval = ref(null)
const copyFallbackUrl = ref('')
const copyInput = ref(null)

const items = computed(() => (listing.value && listing.value.statusPages) || [])

const removalMessage = computed(() => (pendingRemoval.value ? `Status page ${pendingRemoval.value.slug} will be removed and ${pendingRemoval.value.path} will stop responding.` : ''))

const stateLabel = (item) => {
  if (item.conflict) {
    return 'Conflicts with YAML'
  }
  if (item.error) {
    return 'Invalid'
  }
  if (!item.enabled) {
    return 'Disabled'
  }
  return item.published ? 'Published' : 'Not published'
}

const stateClass = (item) => {
  if (item.conflict) {
    return 'text-amber-700 dark:text-amber-400'
  }
  if (item.error) {
    return 'text-red-700 dark:text-red-400'
  }
  return item.published ? 'text-green-700 dark:text-green-400' : 'text-muted-foreground dark:text-gray-500'
}

const load = async () => {
  error.value = ''
  try {
    const { data } = await statusPagesApi.list()
    listing.value = data
  } catch (e) {
    error.value = describeStatusPageError(e)
  } finally {
    loading.value = false
  }
}

const edit = (item) => {
  router.push({ name: 'AdminStatusPageEdit', params: { slug: item.slug } })
}

const copyLink = async (item) => {
  const url = `${window.location.origin}${item.path}`
  notice.value = ''
  copyFallbackUrl.value = ''
  try {
    if (!navigator.clipboard) {
      throw new Error('clipboard unavailable')
    }
    await navigator.clipboard.writeText(url)
    notice.value = `Link copied: ${url}`
  } catch (e) {
    copyFallbackUrl.value = url
    await nextTick()
    if (copyInput.value) {
      copyInput.value.focus()
      copyInput.value.select()
    }
  }
}

const toggle = async (item) => {
  busySlug.value = item.slug
  notice.value = ''
  error.value = ''
  try {
    await statusPagesApi.setEnabled(item.slug, !item.enabled, item.version)
    notice.value = `Status page ${item.slug} ${item.enabled ? 'disabled' : 'enabled'}.`
  } catch (e) {
    error.value = describeStatusPageError(e)
  } finally {
    busySlug.value = ''
    await load()
  }
}

const confirmRemoval = async () => {
  const item = pendingRemoval.value
  pendingRemoval.value = null
  if (!item) {
    return
  }
  busySlug.value = item.slug
  notice.value = ''
  error.value = ''
  try {
    await statusPagesApi.remove(item.slug, item.version)
    notice.value = `Status page ${item.slug} removed.`
  } catch (e) {
    error.value = describeStatusPageError(e)
  } finally {
    busySlug.value = ''
    await load()
  }
}

onMounted(load)
</script>
