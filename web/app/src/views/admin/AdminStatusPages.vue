<template>
  <AdminListLayout title="Status pages" description="Public pages, open without login at /status/<slug>" active="status-pages">
    <template #actions>
      <router-link to="/" class="inline-flex h-9 items-center border border-input bg-background px-3 text-sm font-medium hover:bg-accent dark:border-gray-700 dark:hover:bg-gray-800">Dashboard</router-link>
      <Button size="sm" data-testid="admin-new-status-page" @click="router.push({ name: 'AdminStatusPageNew' })">New status page</Button>
    </template>

    <template #notices>
      <div v-if="listing && !listing.publicationEnabled" role="status" data-testid="status-pages-disabled" class="mt-3 shrink-0 border border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100">
        Status pages are disabled in the configuration file (<code>status-pages.enabled: false</code>): no page is published, but you can still edit and preview them.
      </div>
      <div v-if="listing && listing.managedUnavailable" role="alert" data-testid="status-pages-unavailable" class="mt-3 shrink-0 border border-red-300 bg-red-50 px-4 py-2 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">
        The status pages managed through the web could not be loaded: only the ones from the configuration file are published until the next reload.
      </div>
      <div v-if="listing && listing.sharedRateLimitWarning" role="status" data-testid="status-pages-shared-limit" class="mt-3 shrink-0 border border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-100">
        Visits come from <code>{{ listing.sharedRateLimitWarning }}</code> with X-Forwarded-For, but this IP is not in <code>status-pages.trusted-proxies</code>: all visitors share the same rate limit.
      </div>
      <div v-if="notice" role="status" data-testid="admin-notice" class="mt-3 shrink-0 border border-green-300 bg-green-50 px-4 py-2 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200">{{ notice }}</div>
      <div v-if="error" role="alert" data-testid="admin-error" class="mt-3 shrink-0 border border-red-300 bg-red-50 px-4 py-2 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">{{ error }}</div>
      <div v-if="copyFallbackUrl" class="mt-3 shrink-0 border bg-card px-4 py-2 text-sm dark:border-gray-700 dark:bg-gray-900">
        <label class="block text-foreground dark:text-gray-200">Copy the address of the page:
          <input ref="copyInput" :value="copyFallbackUrl" readonly class="mt-1 h-9 w-full border border-input bg-background px-2 font-mono text-sm dark:border-gray-700" data-testid="status-page-copy-fallback" @focus="$event.target.select()" />
        </label>
      </div>
    </template>

    <div v-if="loading" class="py-12 flex justify-center"><Loading /></div>
    <!-- Fork: fixed layout so that the width of the table never depends on the content, and cards below md -->
    <table v-else class="hidden w-full table-fixed text-sm md:table" data-testid="status-pages-table">
      <thead class="sticky top-0 z-10 bg-gray-50 text-left text-muted-foreground shadow-[0_1px_0_0_rgb(229,231,235)] dark:bg-gray-800 dark:text-gray-400 dark:shadow-[0_1px_0_0_rgb(55,65,81)]">
        <tr>
          <th class="w-[20%] px-3 py-2 font-medium">Slug</th>
          <th class="px-3 py-2 font-medium">Title</th>
          <th class="hidden w-[8%] px-3 py-2 font-medium lg:table-cell">Source</th>
          <th class="w-[12%] px-3 py-2 font-medium">Status</th>
          <th class="hidden w-[10%] px-3 py-2 font-medium lg:table-cell">Endpoints</th>
          <th class="w-[25rem] px-3 py-2 font-medium text-right">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="items.length === 0">
          <td colspan="6" class="px-3 py-8 text-center text-muted-foreground dark:text-gray-400">No status pages yet.</td>
        </tr>
        <tr v-for="item in items" :key="`${item.origin}-${item.slug}`" class="border-t hover:bg-muted/40 dark:border-gray-700 dark:hover:bg-gray-800/50" :data-testid="`status-page-row-${item.origin}-${item.slug}`">
          <td class="px-3 py-1.5 font-mono text-xs text-foreground dark:text-gray-100"><span class="block truncate" :title="item.slug">{{ item.slug }}</span></td>
          <td class="px-3 py-1.5 font-medium text-foreground dark:text-gray-100"><span class="block truncate" :title="item.title || ''">{{ item.title || '—' }}</span></td>
          <td class="hidden px-3 py-1.5 lg:table-cell">
            <span :class="['border px-1.5 py-0.5 text-xs', item.origin === 'admin' ? 'border-blue-300 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-200' : 'border-gray-300 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300']">{{ item.origin === 'admin' ? 'Web' : 'YAML' }}</span>
          </td>
          <td class="px-3 py-1.5">
            <span :class="stateClass(item)" :title="item.conflictOrigin || item.error || ''">{{ stateLabel(item) }}</span>
          </td>
          <td class="hidden px-3 py-1.5 lg:table-cell">{{ item.endpoints }}</td>
          <td class="px-3 py-1.5 whitespace-nowrap text-right">
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

    <!-- Fork: below md the table gives way to one card per status page, with every field and the same actions -->
    <div v-if="!loading" class="divide-y md:hidden dark:divide-gray-700">
      <p v-if="items.length === 0" class="px-3 py-8 text-center text-sm text-muted-foreground dark:text-gray-400">No status pages yet.</p>
      <div v-for="item in items" :key="`card-${item.origin}-${item.slug}`" class="px-3 py-2.5" :data-testid="`status-page-card-${item.origin}-${item.slug}`">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="truncate font-medium text-foreground dark:text-gray-100" :title="item.title || ''">{{ item.title || '—' }}</p>
            <p class="truncate font-mono text-xs text-muted-foreground dark:text-gray-400" :title="item.slug">{{ item.slug }}</p>
          </div>
          <span :class="['shrink-0 text-xs', stateClass(item)]" :title="item.conflictOrigin || item.error || ''">{{ stateLabel(item) }}</span>
        </div>
        <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground dark:text-gray-400">
          <span>{{ item.origin === 'admin' ? 'Web' : 'YAML' }}</span>
          <span>{{ item.endpoints }} {{ item.endpoints === 1 ? 'endpoint' : 'endpoints' }}</span>
        </div>
        <div class="mt-1 flex flex-wrap items-center gap-1">
            <a :href="item.path" target="_blank" rel="noopener" class="inline-flex h-9 items-center px-3 text-sm font-medium hover:bg-accent dark:hover:bg-gray-800" :data-testid="`status-page-open-${item.slug}`">Open</a>
            <Button variant="ghost" size="sm" :data-testid="`status-page-copy-${item.slug}`" @click="copyLink(item)">Copy link</Button>
            <Button variant="ghost" size="sm" :data-testid="`status-page-edit-${item.slug}`" @click="edit(item)">{{ item.origin === 'admin' ? 'Edit' : 'View' }}</Button>
            <template v-if="item.origin === 'admin'">
              <Button variant="ghost" size="sm" :disabled="busySlug === item.slug || item.conflict || Boolean(item.error)" :data-testid="`status-page-toggle-${item.slug}`" @click="toggle(item)">{{ item.enabled ? 'Disable' : 'Enable' }}</Button>
              <Button variant="ghost" size="sm" class="text-red-600 dark:text-red-400" :disabled="busySlug === item.slug" :data-testid="`status-page-remove-${item.slug}`" @click="pendingRemoval = item">Remove</Button>
            </template>
        </div>
      </div>
    </div>

    <template #footer>
      {{ items.length }} {{ items.length === 1 ? 'status page' : 'status pages' }}<span v-if="publishedCount"> · {{ publishedCount }} published</span>
    </template>

    <template #overlay>
      <ConfirmDialog
        :open="pendingRemoval !== null"
        title="Remove status page"
        :message="removalMessage"
        confirm-label="Remove"
        @confirm="confirmRemoval"
        @cancel="pendingRemoval = null"
      />
    </template>
  </AdminListLayout>
</template>

<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import Loading from '@/components/Loading.vue'
import ConfirmDialog from '@/components/admin/ConfirmDialog.vue'
import AdminListLayout from '@/components/admin/AdminListLayout.vue'
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
const publishedCount = computed(() => items.value.filter((item) => item.published).length)

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
