<template>
  <div class="container mx-auto px-4 py-8 max-w-5xl">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground dark:text-gray-100">{{ title }}</h1>
        <p v-if="isEdit" class="mt-1 font-mono text-sm text-muted-foreground dark:text-gray-400">
          {{ endpointKey }}<span v-if="version"> · versão {{ version }}</span>
        </p>
      </div>
      <Button variant="outline" data-testid="admin-back" @click="goBack">Voltar</Button>
    </div>

    <div v-if="loading" class="py-12 flex justify-center"><Loading /></div>
    <template v-else>
      <div v-if="error" role="alert" data-testid="admin-error" class="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200">
        <p class="whitespace-pre-line">{{ error }}</p>
        <Button v-if="versionConflict" variant="outline" size="sm" class="mt-2" data-testid="admin-reload" @click="reloadCurrentVersion">Recarregar versão atual</Button>
      </div>
      <div v-if="success" role="status" data-testid="admin-success" class="mb-4 border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/30 dark:text-green-200">{{ success }}</div>
      <div v-if="readOnly" class="mb-4 border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
        Este endpoint é definido no arquivo de configuração e só pode ser visualizado.
      </div>

      <div v-if="!readOnly" class="mb-4 flex border-b dark:border-gray-700" role="tablist">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          role="tab"
          :aria-selected="mode === tab.value"
          :data-testid="`admin-mode-${tab.value}`"
          :class="['-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors', mode === tab.value ? 'border-primary text-foreground dark:border-gray-100 dark:text-gray-100' : 'border-transparent text-muted-foreground hover:text-foreground dark:text-gray-400 dark:hover:text-gray-100']"
          @click="switchMode(tab.value)"
        >{{ tab.label }}</button>
      </div>

      <div v-if="mode === 'form' && !readOnly" class="space-y-6 border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block text-sm font-medium text-foreground dark:text-gray-200">Nome
            <Input v-model="form.name" :disabled="isEdit" class="mt-1 dark:border-gray-700" data-testid="admin-field-name" />
          </label>
          <label class="block text-sm font-medium text-foreground dark:text-gray-200">Grupo
            <Input v-model="form.group" :disabled="isEdit" class="mt-1 dark:border-gray-700" data-testid="admin-field-group" />
          </label>
          <label class="block text-sm font-medium text-foreground dark:text-gray-200 sm:col-span-2">URL
            <Input v-model="form.url" placeholder="https://exemplo.com/health" class="mt-1 font-mono dark:border-gray-700" data-testid="admin-field-url" />
          </label>
          <div class="text-sm font-medium text-foreground dark:text-gray-200">Método
            <Select v-model="form.method" :options="methodOptions" placeholder="GET (padrão)" class="mt-1" />
          </div>
          <label class="block text-sm font-medium text-foreground dark:text-gray-200">Intervalo
            <Input v-model="form.interval" placeholder="1m (padrão)" class="mt-1 dark:border-gray-700" data-testid="admin-field-interval" />
          </label>
          <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200 sm:col-span-2">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" data-testid="admin-field-enabled" />
            Habilitado
          </label>
        </div>

        <section>
          <h2 class="mb-2 text-sm font-semibold text-foreground dark:text-gray-200">Condições</h2>
          <div v-for="(condition, index) in form.conditions" :key="`condition-${index}`" class="mb-2 flex gap-2">
            <Input v-model="form.conditions[index]" placeholder="[STATUS] == 200" class="font-mono dark:border-gray-700" :data-testid="`admin-field-condition-${index}`" />
            <Button variant="ghost" size="sm" aria-label="Remover condição" @click="form.conditions.splice(index, 1)">✕</Button>
          </div>
          <Button variant="outline" size="sm" data-testid="admin-add-condition" @click="form.conditions.push('')">Adicionar condição</Button>
        </section>

        <section>
          <h2 class="mb-2 text-sm font-semibold text-foreground dark:text-gray-200">Headers</h2>
          <div v-for="(header, index) in form.headers" :key="`header-${index}`" class="mb-2 grid grid-cols-[1fr_2fr_auto] gap-2">
            <Input v-model="header.name" placeholder="Nome" class="dark:border-gray-700" :data-testid="`admin-field-header-name-${index}`" />
            <Input v-model="header.value" placeholder="Valor" class="font-mono dark:border-gray-700" :data-testid="`admin-field-header-value-${index}`" />
            <Button variant="ghost" size="sm" aria-label="Remover header" @click="form.headers.splice(index, 1)">✕</Button>
          </div>
          <Button variant="outline" size="sm" data-testid="admin-add-header" @click="form.headers.push({ name: '', value: '' })">Adicionar header</Button>
        </section>

        <section>
          <h2 class="mb-2 text-sm font-semibold text-foreground dark:text-gray-200">Alertas</h2>
          <p v-if="alertTypeOptions.length === 0" class="text-sm text-muted-foreground dark:text-gray-400">Nenhum provedor de alerta configurado.</p>
          <div v-for="(alert, index) in form.alerts" :key="`alert-${index}`" class="mb-3 grid gap-2 border p-3 sm:grid-cols-2 dark:border-gray-700">
            <div class="text-sm text-foreground dark:text-gray-200">Tipo
              <Select v-model="alert.type" :options="alertTypeOptions" placeholder="Selecione" class="mt-1" />
            </div>
            <label class="block text-sm text-foreground dark:text-gray-200">Descrição
              <Input v-model="alert.description" class="mt-1 dark:border-gray-700" />
            </label>
            <label class="block text-sm text-foreground dark:text-gray-200">Falhas para disparar
              <Input v-model="alert.failureThreshold" type="number" min="1" placeholder="padrão do provedor" class="mt-1 dark:border-gray-700" />
            </label>
            <label class="block text-sm text-foreground dark:text-gray-200">Sucessos para resolver
              <Input v-model="alert.successThreshold" type="number" min="1" placeholder="padrão do provedor" class="mt-1 dark:border-gray-700" />
            </label>
            <label class="flex items-center gap-2 text-sm text-foreground dark:text-gray-200">
              <input v-model="alert.sendOnResolved" type="checkbox" class="h-4 w-4 accent-gray-900 dark:accent-gray-100" />
              Notificar resolução
            </label>
            <div class="text-right">
              <Button variant="ghost" size="sm" class="text-red-600 dark:text-red-400" @click="form.alerts.splice(index, 1)">Remover alerta</Button>
            </div>
          </div>
          <Button v-if="alertTypeOptions.length > 0" variant="outline" size="sm" data-testid="admin-add-alert" @click="addAlert">Adicionar alerta</Button>
        </section>
      </div>

      <div v-else class="border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <textarea
          v-model="yamlText"
          :readonly="readOnly"
          spellcheck="false"
          data-testid="admin-yaml"
          class="min-h-[24rem] w-full border border-input bg-background p-3 font-mono text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100"
        ></textarea>
      </div>

      <div v-if="!readOnly" class="mt-6 flex flex-wrap gap-2">
        <Button variant="outline" :disabled="busy" data-testid="admin-validate" @click="validate">Validar</Button>
        <Button variant="secondary" :disabled="busy" data-testid="admin-test" @click="test">Testar</Button>
        <Button :disabled="busy" data-testid="admin-save" @click="save">Salvar</Button>
      </div>

      <div v-if="testResult" data-testid="admin-test-result" class="mt-6 border bg-card p-6 dark:border-gray-700 dark:bg-gray-900">
        <h2 class="mb-1 text-lg font-semibold text-foreground dark:text-gray-100">
          Resultado do teste:
          <span :class="testResult.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">{{ testResult.success ? 'sucesso' : 'falha' }}</span>
        </h2>
        <p class="mb-3 text-sm text-muted-foreground dark:text-gray-400">
          Duração: {{ testResult.durationMs }} ms<span v-if="testResult.status"> · HTTP {{ testResult.status }}</span>
        </p>
        <ul class="space-y-1 font-mono text-sm">
          <li
            v-for="(condition, index) in testResult.conditionResults"
            :key="index"
            :class="condition.success ? 'text-green-700 dark:text-green-400' : 'text-red-700 dark:text-red-400'"
          >{{ condition.success ? '✓' : '✗' }} {{ condition.condition }}</li>
        </ul>
        <ul v-if="testResult.errors && testResult.errors.length" class="mt-3 list-disc pl-5 text-sm text-red-700 dark:text-red-400">
          <li v-for="(message, index) in testResult.errors" :key="index">{{ message }}</li>
        </ul>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import Loading from '@/components/Loading.vue'
import { adminApi, describeAdminError, jsonPayload, yamlPayload } from '@/utils/adminApi'
import { toYaml } from '@/utils/yaml'

const props = defineProps({
  endpointKey: { type: String, default: '' },
})

const router = useRouter()

const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'].map((method) => ({ label: method, value: method }))
const tabs = [
  { label: 'Formulário', value: 'form' },
  { label: 'YAML', value: 'yaml' },
]

const loading = ref(true)
const busy = ref(false)
const error = ref('')
const success = ref('')
const versionConflict = ref(false)
const testResult = ref(null)
const mode = ref('form')
const yamlText = ref('')
const version = ref(0)
const source = ref('admin')
const alertTypes = ref([])
// Keys of the definition that the form does not edit (e.g. client, dns) are kept from this document
const baseDocument = ref({})

const emptyForm = () => ({
  name: '',
  group: '',
  url: '',
  method: '',
  interval: '',
  enabled: true,
  conditions: ['[STATUS] == 200'],
  headers: [],
  alerts: [],
})

const form = reactive(emptyForm())

const isEdit = computed(() => props.endpointKey !== '')
const readOnly = computed(() => source.value === 'config')
const title = computed(() => {
  if (!isEdit.value) {
    return 'Novo endpoint'
  }
  return readOnly.value ? 'Endpoint do arquivo de configuração' : 'Editar endpoint'
})
const alertTypeOptions = computed(() => alertTypes.value.map((type) => ({ label: type, value: type })))

const clearMessages = () => {
  error.value = ''
  success.value = ''
  versionConflict.value = false
}

const formFromDocument = (document) => {
  Object.assign(form, {
    name: document.name || '',
    group: document.group || '',
    url: document.url || '',
    method: document.method || '',
    interval: document.interval || '',
    enabled: document.enabled !== false,
    conditions: Array.isArray(document.conditions) ? document.conditions.map(String) : [],
    headers: document.headers ? Object.entries(document.headers).map(([name, value]) => ({ name, value: String(value) })) : [],
    alerts: Array.isArray(document.alerts)
      ? document.alerts.map((alert) => ({
        type: alert.type || '',
        description: alert.description || '',
        failureThreshold: alert['failure-threshold'] ? String(alert['failure-threshold']) : '',
        successThreshold: alert['success-threshold'] ? String(alert['success-threshold']) : '',
        sendOnResolved: alert['send-on-resolved'] === true,
        original: alert,
      }))
      : [],
  })
}

const setOrDelete = (document, key, value) => {
  if (value === '' || value === undefined || value === null) {
    delete document[key]
  } else {
    document[key] = value
  }
}

const documentFromForm = () => {
  const document = JSON.parse(JSON.stringify(baseDocument.value || {}))
  setOrDelete(document, 'name', form.name.trim())
  setOrDelete(document, 'group', form.group.trim())
  setOrDelete(document, 'url', form.url.trim())
  setOrDelete(document, 'method', form.method)
  setOrDelete(document, 'interval', form.interval.trim())
  if (form.enabled) {
    delete document.enabled
  } else {
    document.enabled = false
  }
  const conditions = form.conditions.map((condition) => condition.trim()).filter(Boolean)
  setOrDelete(document, 'conditions', conditions.length ? conditions : null)
  const headers = {}
  form.headers.filter((header) => header.name.trim()).forEach((header) => {
    headers[header.name.trim()] = header.value
  })
  setOrDelete(document, 'headers', Object.keys(headers).length ? headers : null)
  const alerts = form.alerts.filter((alert) => alert.type).map((alert) => {
    const result = { ...(alert.original || {}), type: alert.type }
    setOrDelete(result, 'description', alert.description.trim())
    setOrDelete(result, 'failure-threshold', Number(alert.failureThreshold) > 0 ? Number(alert.failureThreshold) : null)
    setOrDelete(result, 'success-threshold', Number(alert.successThreshold) > 0 ? Number(alert.successThreshold) : null)
    setOrDelete(result, 'send-on-resolved', alert.sendOnResolved ? true : null)
    return result
  })
  setOrDelete(document, 'alerts', alerts.length ? alerts : null)
  return document
}

const currentPayload = () => (mode.value === 'form' ? jsonPayload(documentFromForm()) : yamlPayload(yamlText.value))

const addAlert = () => {
  form.alerts.push({ type: alertTypes.value[0] || '', description: '', failureThreshold: '', successThreshold: '', sendOnResolved: false, original: null })
}

const loadDetail = async () => {
  const { data } = await adminApi.get(props.endpointKey)
  source.value = data.source
  version.value = data.version || 0
  baseDocument.value = data.definition.json || {}
  yamlText.value = data.definition.yaml || ''
  formFromDocument(baseDocument.value)
  if (readOnly.value) {
    mode.value = 'yaml'
  }
}

const switchMode = async (target) => {
  if (target === mode.value) {
    return
  }
  clearMessages()
  if (target === 'yaml') {
    yamlText.value = `${toYaml(documentFromForm())}\n`
    mode.value = 'yaml'
    return
  }
  busy.value = true
  try {
    const { data } = await adminApi.parse(yamlPayload(yamlText.value))
    baseDocument.value = data.json || {}
    formFromDocument(baseDocument.value)
    mode.value = 'form'
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busy.value = false
  }
}

const validate = async () => {
  clearMessages()
  busy.value = true
  try {
    await adminApi.validate(currentPayload(), props.endpointKey)
    success.value = 'Definição válida.'
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busy.value = false
  }
}

const test = async () => {
  clearMessages()
  testResult.value = null
  busy.value = true
  try {
    const { data } = await adminApi.test(currentPayload(), props.endpointKey)
    testResult.value = data
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    busy.value = false
  }
}

const save = async () => {
  clearMessages()
  busy.value = true
  try {
    if (isEdit.value) {
      await adminApi.update(props.endpointKey, currentPayload(), version.value)
    } else {
      await adminApi.create(currentPayload())
    }
    router.push({ name: 'AdminEndpoints' })
  } catch (e) {
    error.value = describeAdminError(e)
    versionConflict.value = e.status === 412
  } finally {
    busy.value = false
  }
}

const reloadCurrentVersion = async () => {
  clearMessages()
  try {
    await loadDetail()
    success.value = 'Versão atual carregada.'
  } catch (e) {
    error.value = describeAdminError(e)
  }
}

const goBack = () => {
  router.push({ name: 'AdminEndpoints' })
}

onMounted(async () => {
  try {
    const { data } = await adminApi.metadata()
    alertTypes.value = (data && data.alertTypes) || []
  } catch (e) {
    alertTypes.value = []
  }
  try {
    if (isEdit.value) {
      await loadDetail()
    }
  } catch (e) {
    error.value = describeAdminError(e)
  } finally {
    loading.value = false
  }
})
</script>
