<template>
  <div class="container mx-auto px-4 py-8 max-w-7xl">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground dark:text-gray-100">Administração de endpoints</h1>
        <p class="text-muted-foreground dark:text-gray-400 mt-1">Endpoints cadastrados pela web e endpoints do arquivo de configuração</p>
      </div>
      <div class="flex gap-2">
        <router-link to="/" class="inline-flex h-10 items-center border border-input bg-background px-4 text-sm font-medium hover:bg-accent dark:border-gray-700 dark:hover:bg-gray-800">Dashboard</router-link>
        <Button data-testid="admin-new-endpoint" @click="router.push({ name: 'AdminEndpointNew' })">Novo endpoint</Button>
      </div>
    </div>

    <Input v-model="search" placeholder="Buscar por nome, grupo ou URL" class="mb-4 dark:border-gray-700" data-testid="admin-search" />

    <div v-if="notice" role="status" data-testid="admin-notice" class="mb-4 border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-950 dark:text-green-200">{{ notice }}</div>
    <div v-if="error" role="alert" data-testid="admin-error" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-950 dark:text-red-200">{{ error }}</div>

    <div v-if="loading" class="py-12 flex justify-center"><Loading /></div>
    <div v-else class="overflow-x-auto border bg-card dark:border-gray-700 dark:bg-gray-900">
      <table class="w-full text-sm" data-testid="admin-table">
        <thead class="bg-muted/50 text-left text-muted-foreground dark:bg-gray-800 dark:text-gray-400">
          <tr>
            <th class="px-3 py-2 font-medium">Nome</th>
            <th class="px-3 py-2 font-medium">Grupo</th>
            <th class="px-3 py-2 font-medium">Tipo</th>
            <th class="px-3 py-2 font-medium">URL</th>
            <th class="px-3 py-2 font-medium">Intervalo</th>
            <th class="px-3 py-2 font-medium">Estado</th>
            <th class="px-3 py-2 font-medium">Origem</th>
            <th class="px-3 py-2 font-medium text-right">Ações</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="filteredItems.length === 0">
            <td colspan="8" class="px-3 py-8 text-center text-muted-foreground dark:text-gray-400">Nenhum endpoint encontrado.</td>
          </tr>
          <tr v-for="item in filteredItems" :key="item.key" class="border-t dark:border-gray-700" :data-testid="`admin-row-${item.key}`">
            <td class="px-3 py-2 font-medium text-foreground dark:text-gray-100">
              {{ item.name }}
              <span v-if="item.conflict" :title="item.conflictOrigin" class="ml-2 border border-amber-300 bg-amber-50 px-1.5 py-0.5 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-200">Conflito com o YAML</span>
              <span v-else-if="item.error" :title="item.error" class="ml-2 border border-red-300 bg-red-50 px-1.5 py-0.5 text-xs text-red-800 dark:border-red-800 dark:bg-red-950 dark:text-red-200">Inválido</span>
            </td>
            <td class="px-3 py-2 text-muted-foreground dark:text-gray-400">{{ item.group }}</td>
            <td class="px-3 py-2 uppercase text-muted-foreground dark:text-gray-400">{{ item.type }}</td>
            <td class="px-3 py-2 max-w-xs truncate font-mono text-xs" :title="item.url">{{ item.url }}</td>
            <td class="px-3 py-2">{{ item.interval }}</td>
            <td class="px-3 py-2">
              <span :class="item.enabled ? 'text-green-700 dark:text-green-400' : 'text-muted-foreground dark:text-gray-500'">{{ item.enabled ? 'Habilitado' : 'Desabilitado' }}</span>
            </td>
            <td class="px-3 py-2">
              <span :class="['border px-1.5 py-0.5 text-xs', item.source === 'admin' ? 'border-blue-300 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-200' : 'border-gray-300 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300']">{{ item.source === 'admin' ? 'Web' : 'YAML' }}</span>
            </td>
            <td class="px-3 py-2 whitespace-nowrap text-right">
              <Button variant="ghost" size="sm" :data-testid="`admin-open-${item.key}`" @click="open(item)">{{ item.source === 'admin' ? 'Editar' : 'Ver' }}</Button>
              <template v-if="item.source === 'admin'">
                <Button variant="ghost" size="sm" :disabled="busyKey === item.key || item.conflict" :data-testid="`admin-toggle-${item.key}`" @click="toggle(item)">{{ item.enabled ? 'Desabilitar' : 'Habilitar' }}</Button>
                <Button variant="ghost" size="sm" class="text-red-600 dark:text-red-400" :disabled="busyKey === item.key" :data-testid="`admin-remove-${item.key}`" @click="pendingRemoval = item">Remover</Button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmDialog
      :open="pendingRemoval !== null"
      title="Remover endpoint"
      :message="removalMessage"
      confirm-label="Remover"
      @confirm="confirmRemoval"
      @cancel="pendingRemoval = null"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Loading from '@/components/Loading.vue'
import ConfirmDialog from '@/components/admin/ConfirmDialog.vue'
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

const removalMessage = computed(() => {
  if (!pendingRemoval.value) {
    return ''
  }
  return `O endpoint ${pendingRemoval.value.key} e todo o seu histórico serão apagados.\nAlertas disparados não serão notificados aos provedores de alerta.`
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
    notice.value = `Endpoint ${item.key} ${item.enabled ? 'desabilitado' : 'habilitado'}.`
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
      ? `Endpoint ${item.key} removido. ${triggeredAlerts} alerta(s) disparado(s) não foram notificados aos provedores.`
      : `Endpoint ${item.key} removido.`
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busyKey.value = ''
    await load()
  }
}

onMounted(load)
</script>
