<template>
  <AdminListLayout title="Endpoint administration" description="Endpoints managed through the web and endpoints from the configuration file" active="endpoints">
    <template #actions>
      <router-link to="/" class="inline-flex h-9 items-center border border-input bg-background px-3 text-sm font-medium hover:bg-accent dark:border-gray-700 dark:hover:bg-gray-800">Dashboard</router-link>
      <Button size="sm" data-testid="admin-new-endpoint" @click="router.push({ name: 'AdminEndpointNew' })">New endpoint</Button>
    </template>

    <template #notices>
      <div v-if="notice" role="status" data-testid="admin-notice" class="mt-3 shrink-0 border border-green-300 bg-green-50 px-4 py-2 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200">{{ notice }}</div>
      <div v-if="error" role="alert" data-testid="admin-error" class="mt-3 shrink-0 border border-red-300 bg-red-50 px-4 py-2 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">{{ error }}</div>
    </template>

    <template #toolbar>
      <Input v-model="search" placeholder="Search by name, group or URL" class="h-9 max-w-md dark:border-gray-700" data-testid="admin-search" />
    </template>

    <div v-if="loading" class="py-12 flex justify-center"><Loading /></div>
    <!-- Fork: fixed layout so that the width of the table never depends on the content, and cards below md -->
    <table v-else class="hidden w-full table-fixed text-sm md:table" data-testid="admin-table">
      <thead class="sticky top-0 z-10 bg-gray-50 text-left text-muted-foreground shadow-[0_1px_0_0_rgb(229,231,235)] dark:bg-gray-800 dark:text-gray-400 dark:shadow-[0_1px_0_0_rgb(55,65,81)]">
        <tr>
          <th class="w-[22%] px-3 py-2 font-medium">Name</th>
          <th class="w-[12%] px-3 py-2 font-medium">Group</th>
          <th class="w-[9%] px-3 py-2 font-medium">Type</th>
          <th class="px-3 py-2 font-medium">URL</th>
          <th class="hidden w-[8%] px-3 py-2 font-medium lg:table-cell">Interval</th>
          <th class="w-[9%] px-3 py-2 font-medium">Status</th>
          <th class="hidden w-[8%] px-3 py-2 font-medium lg:table-cell">Source</th>
          <th class="w-[15rem] px-3 py-2 font-medium text-right">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="filteredItems.length === 0">
          <td colspan="8" class="px-3 py-8 text-center text-muted-foreground dark:text-gray-400">No endpoints found.</td>
        </tr>
        <tr v-for="item in filteredItems" :key="item.key" class="border-t hover:bg-muted/40 dark:border-gray-700 dark:hover:bg-gray-800/50" :data-testid="`admin-row-${item.key}`">
          <td class="px-3 py-1.5 font-medium text-foreground dark:text-gray-100">
            <span class="block truncate" :title="item.name">{{ item.name }}</span>
            <!-- The warning is on a line of its own so that it does not add width to the cell -->
            <span v-if="item.conflict" :title="item.conflictOrigin" class="mt-0.5 inline-block border border-amber-300 bg-amber-50 px-1.5 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-200">Conflicts with YAML</span>
            <span v-else-if="item.error" :title="item.error" class="mt-0.5 inline-block border border-red-300 bg-red-50 px-1.5 text-xs text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">Invalid</span>
          </td>
          <td class="px-3 py-1.5 text-muted-foreground dark:text-gray-400"><span class="block truncate" :title="item.group">{{ item.group }}</span></td>
          <td class="px-3 py-1.5 uppercase text-muted-foreground dark:text-gray-400">
            {{ item.type }}
            <span v-if="item.acceptsPush && item.type !== 'PUSH'" title="Also receives push" class="ml-1 border border-violet-300 bg-violet-50 px-1 py-0.5 text-xs normal-case text-violet-800 dark:border-violet-800 dark:bg-violet-900/30 dark:text-violet-200" :data-testid="`admin-accepts-push-${item.key}`">+ push</span>
          </td>
          <td class="px-3 py-1.5 font-mono text-xs" :title="item.url"><span class="block truncate">{{ item.url || (item.type === 'PUSH' ? '—' : '') }}</span></td>
          <td class="hidden px-3 py-1.5 lg:table-cell">{{ item.interval }}</td>
          <td class="px-3 py-1.5">
            <span :class="item.enabled ? 'text-green-700 dark:text-green-400' : 'text-muted-foreground dark:text-gray-500'">{{ item.enabled ? 'Enabled' : 'Disabled' }}</span>
          </td>
          <td class="hidden px-3 py-1.5 lg:table-cell">
            <span :class="['border px-1.5 py-0.5 text-xs', item.source === 'admin' ? 'border-blue-300 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-200' : 'border-gray-300 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300']">{{ item.source === 'admin' ? 'Web' : 'YAML' }}</span>
          </td>
          <td class="px-3 py-1.5 whitespace-nowrap text-right">
            <Button variant="ghost" size="sm" :data-testid="`admin-open-${item.key}`" @click="open(item)">{{ item.source === 'admin' ? 'Edit' : 'View' }}</Button>
            <template v-if="item.source === 'admin'">
              <Button variant="ghost" size="sm" :disabled="busyKey === item.key || item.conflict" :data-testid="`admin-toggle-${item.key}`" @click="toggle(item)">{{ item.enabled ? 'Disable' : 'Enable' }}</Button>
              <Button variant="ghost" size="sm" class="text-red-600 dark:text-red-400" :disabled="busyKey === item.key" :data-testid="`admin-remove-${item.key}`" @click="pendingRemoval = item">Remove</Button>
            </template>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Fork: below md the table gives way to one card per endpoint, with every field and the same actions -->
    <div v-if="!loading" class="divide-y md:hidden dark:divide-gray-700">
      <p v-if="filteredItems.length === 0" class="px-3 py-8 text-center text-sm text-muted-foreground dark:text-gray-400">No endpoints found.</p>
      <div v-for="item in filteredItems" :key="`card-${item.key}`" class="px-3 py-2.5" :data-testid="`admin-card-${item.key}`">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="truncate font-medium text-foreground dark:text-gray-100" :title="item.name">{{ item.name }}</p>
            <p class="truncate font-mono text-xs text-muted-foreground dark:text-gray-400" :title="item.url">{{ item.url || (item.type === 'PUSH' ? '—' : '') }}</p>
          </div>
          <span :class="['shrink-0 text-xs', item.enabled ? 'text-green-700 dark:text-green-400' : 'text-muted-foreground dark:text-gray-500']">{{ item.enabled ? 'Enabled' : 'Disabled' }}</span>
        </div>
        <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground dark:text-gray-400">
          <span v-if="item.group" class="truncate">{{ item.group }}</span>
          <span class="uppercase">{{ item.type }}</span>
          <span v-if="item.acceptsPush && item.type !== 'PUSH'" class="normal-case">+ push</span>
          <span>{{ item.interval }}</span>
          <span>{{ item.source === 'admin' ? 'Web' : 'YAML' }}</span>
          <span v-if="item.conflict" :title="item.conflictOrigin" class="text-amber-700 dark:text-amber-300">Conflicts with YAML</span>
          <span v-else-if="item.error" :title="item.error" class="text-red-700 dark:text-red-300">Invalid</span>
        </div>
        <div class="mt-1 flex flex-wrap items-center gap-1">
          <Button variant="ghost" size="sm" :data-testid="`admin-open-${item.key}`" @click="open(item)">{{ item.source === 'admin' ? 'Edit' : 'View' }}</Button>
          <template v-if="item.source === 'admin'">
            <Button variant="ghost" size="sm" :disabled="busyKey === item.key || item.conflict" :data-testid="`admin-toggle-${item.key}`" @click="toggle(item)">{{ item.enabled ? 'Disable' : 'Enable' }}</Button>
            <Button variant="ghost" size="sm" class="text-red-600 dark:text-red-400" :disabled="busyKey === item.key" :data-testid="`admin-remove-${item.key}`" @click="pendingRemoval = item">Remove</Button>
          </template>
        </div>
      </div>
    </div>

    <template #footer>
      <span data-testid="admin-count">{{ search.trim() ? `${filteredItems.length} of ${items.length}` : items.length }} {{ items.length === 1 ? 'endpoint' : 'endpoints' }}</span><span v-if="webCount"> · {{ webCount }} managed through the web</span><span v-if="configCount"> · {{ configCount }} from the configuration file</span>
    </template>

    <template #overlay>
      <ConfirmDialog
        :open="pendingRemoval !== null"
        title="Remove endpoint"
        :message="removalMessage"
        confirm-label="Remove"
        @confirm="confirmRemoval"
        @cancel="pendingRemoval = null"
      />
    </template>
  </AdminListLayout>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Loading from '@/components/Loading.vue'
import ConfirmDialog from '@/components/admin/ConfirmDialog.vue'
import AdminListLayout from '@/components/admin/AdminListLayout.vue'
import { adminApi, describeAdminError } from '@/utils/adminApi'

const router = useRouter()

const items = ref([])
const loading = ref(true)
const error = ref('')
const notice = ref('')
const search = ref('')
const busyKey = ref('')
const pendingRemoval = ref(null)

const filteredItems = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) {
    return items.value
  }
  return items.value.filter((item) =>
    [item.name, item.group, item.url].some((value) => (value || '').toLowerCase().includes(query))
  )
})

const webCount = computed(() => items.value.filter((item) => item.source === 'admin').length)
const configCount = computed(() => items.value.length - webCount.value)

const removalMessage = computed(() => {
  if (!pendingRemoval.value) {
    return ''
  }
  return `Endpoint ${pendingRemoval.value.key} and all of its history will be deleted.\nTriggered alerts will not be resolved with the alerting providers.`
})

const load = async () => {
  error.value = ''
  try {
    const { data } = await adminApi.list()
    items.value = data || []
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    loading.value = false
  }
}

const open = (item) => {
  router.push({ name: 'AdminEndpointEdit', params: { endpointKey: item.key } })
}

const toggle = async (item) => {
  busyKey.value = item.key
  notice.value = ''
  error.value = ''
  try {
    await adminApi.setEnabled(item.key, !item.enabled, item.version)
    notice.value = `Endpoint ${item.key} ${item.enabled ? 'disabled' : 'enabled'}.`
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busyKey.value = ''
    await load()
  }
}

const confirmRemoval = async () => {
  const item = pendingRemoval.value
  pendingRemoval.value = null
  if (!item) {
    return
  }
  busyKey.value = item.key
  notice.value = ''
  error.value = ''
  try {
    const { data } = await adminApi.remove(item.key, item.version)
    const triggeredAlerts = (data && data.triggeredAlerts) || 0
    notice.value = triggeredAlerts > 0
      ? `Endpoint ${item.key} removed. ${triggeredAlerts} triggered alert(s) were not resolved with the alerting providers.`
      : `Endpoint ${item.key} removed.`
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busyKey.value = ''
    await load()
  }
}

onMounted(load)
</script>
