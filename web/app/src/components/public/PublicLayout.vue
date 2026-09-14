<template>
  <div class="min-h-screen flex flex-col bg-background text-foreground" data-testid="public-layout">
    <header class="border-b bg-card/50 dark:border-gray-800">
      <div class="container mx-auto px-4 py-3 max-w-5xl flex items-center justify-between gap-4">
        <component
          :is="link ? 'a' : 'div'"
          :href="link || undefined"
          :target="link ? '_blank' : undefined"
          :rel="link ? 'noopener' : undefined"
          class="flex items-center gap-3 min-w-0"
        >
          <img v-if="logo" :src="logo" alt="" class="w-10 h-10 object-contain flex-shrink-0" />
          <img v-else src="@/assets/logo.svg" alt="" class="w-10 h-10 object-contain flex-shrink-0" />
          <span class="text-lg font-semibold truncate">{{ header }}</span>
        </component>
        <button
          type="button"
          class="inline-flex h-9 items-center gap-2 border border-input bg-background px-3 text-sm hover:bg-accent motion-safe:transition-colors dark:border-gray-700 dark:hover:bg-gray-800"
          :aria-label="darkMode ? 'Switch to light mode' : 'Switch to dark mode'"
          data-testid="public-theme-toggle"
          @click="toggleTheme"
        >
          <Sun v-if="darkMode" class="h-4 w-4" aria-hidden="true" />
          <Moon v-else class="h-4 w-4" aria-hidden="true" />
          <span class="hidden sm:inline">{{ darkMode ? 'Light mode' : 'Dark mode' }}</span>
        </button>
      </div>
    </header>
    <main class="flex-1">
      <slot />
    </main>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { Moon, Sun } from 'lucide-vue-next'

const THEME_COOKIE_NAME = 'theme'
const THEME_COOKIE_MAX_AGE = 31536000 // 1 year

const templateValue = (value, placeholder) => (value && value !== placeholder ? value : '')

const logo = computed(() => templateValue(window.config?.logo, '{{ .UI.Logo }}'))
const header = computed(() => templateValue(window.config?.header, '{{ .UI.Header }}') || 'Gatus')
const link = computed(() => templateValue(window.config?.link, '{{ .UI.Link }}') || null)

const wantsDarkMode = () => {
  const themeFromCookie = document.cookie.match(new RegExp(`${THEME_COOKIE_NAME}=(dark|light);?`))?.[1]
  return themeFromCookie === 'dark' || (!themeFromCookie && (window.matchMedia('(prefers-color-scheme: dark)').matches || document.documentElement.classList.contains('dark')))
}

const darkMode = ref(wantsDarkMode())

const toggleTheme = () => {
  const theme = wantsDarkMode() ? 'light' : 'dark'
  document.cookie = `${THEME_COOKIE_NAME}=${theme}; path=/; max-age=${THEME_COOKIE_MAX_AGE}; samesite=strict`
  darkMode.value = theme === 'dark'
  document.documentElement.classList.toggle('dark', darkMode.value)
}
</script>
