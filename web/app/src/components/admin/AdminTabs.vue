<template>
  <nav class="mb-6 flex border-b dark:border-gray-700" aria-label="Administration sections" data-testid="admin-tabs">
    <router-link
      v-for="tab in tabs"
      :key="tab.id"
      :to="{ name: tab.route }"
      :aria-current="active === tab.id ? 'page' : undefined"
      :data-testid="`admin-tab-${tab.id}`"
      :class="[
        '-mb-px border-b-2 px-4 py-2 text-sm font-medium',
        active === tab.id
          ? 'border-foreground text-foreground dark:border-gray-100 dark:text-gray-100'
          : 'border-transparent text-muted-foreground hover:text-foreground dark:text-gray-400 dark:hover:text-gray-100'
      ]"
    >
      {{ tab.label }}
    </router-link>
  </nav>
</template>

<script setup>
defineProps({
  // endpoints or status-pages
  active: { type: String, required: true }
})

const tabs = [
  { id: 'endpoints', label: 'Endpoints', route: 'AdminEndpoints' },
  { id: 'status-pages', label: 'Status pages', route: 'AdminStatusPages' }
]
</script>
