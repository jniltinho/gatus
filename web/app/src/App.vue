<template>
  <div id="global" class="bg-background text-foreground">
    <!-- Waiting for the router: the layout depends on the route -->
    <div v-if="!routerReady" class="flex items-center justify-center min-h-screen">
      <Loading size="lg" />
    </div>

    <!-- Public status pages (fork): no dashboard header, no login screen and no call to /api/v1/config -->
    <PublicLayout v-else-if="isPublic">
      <router-view />
    </PublicLayout>

    <!-- Loading State -->
    <div v-else-if="!retrievedConfig" class="flex items-center justify-center min-h-screen">
      <Loading size="lg" />
    </div>

    <!-- Main App Container -->
    <!-- Fork: the lists of the administration fill the window on larger screens and scroll inside their tables -->
    <div v-else-if="!config || !config.oidc || config.authenticated" :class="['relative', isAdminList && 'md:flex md:h-screen md:flex-col md:overflow-hidden']">
      <!-- Header -->
      <header :class="['border-b bg-card/50 backdrop-blur supports-[backdrop-filter]:bg-card/60', isAdminList && 'md:shrink-0']">
        <div :class="['container mx-auto px-4 max-w-7xl', isAdmin ? 'py-2' : 'py-4']">
          <div class="flex items-center justify-between">
            <!-- Logo and Title -->
            <div class="flex items-center gap-4">
              <component
                :is="link ? 'a' : 'div'"
                :href="link"
                target="_blank"
                :class="['flex items-center gap-3', link && 'hover:opacity-80 transition-opacity']"
              >
                <div :class="['flex items-center justify-center', isAdmin ? 'w-8 h-8' : 'w-12 h-12']">
                  <img
                    v-if="logo"
                    :src="logo"
                    alt="Gatus"
                    class="w-full h-full object-contain"
                  />
                  <img
                    v-else
                    src="./assets/logo.svg"
                    alt="Gatus"
                    class="w-full h-full object-contain"
                  />
                </div>
                <div>
                  <h1 :class="['font-bold tracking-tight', isAdmin ? 'text-lg' : 'text-2xl']">{{ header }}</h1>
                  <p v-if="buttons && buttons.length && !isAdmin" class="text-sm text-muted-foreground">
                    System Monitoring Dashboard
                  </p>
                </div>
              </component>
            </div>

            <!-- Right Side Actions -->
            <div class="flex items-center gap-2">
              <!-- Administration of endpoints (fork) -->
              <router-link
                v-if="showAdminLink"
                to="/admin"
                class="px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground dark:hover:bg-gray-800 transition-colors"
                data-testid="admin-link"
              >
                Admin
              </router-link>
              <!-- Navigation Links (Desktop) -->
              <nav v-if="buttons && buttons.length" class="hidden md:flex items-center gap-1">
                <a
                  v-for="button in buttons"
                  :key="button.name"
                  :href="button.link"
                  target="_blank"
                  class="px-3 py-2 text-sm font-medium rounded-md hover:bg-accent hover:text-accent-foreground transition-colors"
                >
                  {{ button.name }}
                </a>
              </nav>

              <!-- Mobile Menu Button -->
              <Button
                v-if="buttons && buttons.length"
                variant="ghost"
                size="icon"
                class="md:hidden"
                @click="mobileMenuOpen = !mobileMenuOpen"
              >
                <Menu v-if="!mobileMenuOpen" class="h-5 w-5" />
                <X v-else class="h-5 w-5" />
              </Button>
            </div>
          </div>

          <!-- Mobile Navigation -->
          <nav
            v-if="buttons && buttons.length && mobileMenuOpen"
            class="md:hidden mt-4 pt-4 border-t space-y-1"
          >
            <a
              v-for="button in buttons"
              :key="button.name"
              :href="button.link"
              target="_blank"
              class="block px-3 py-2 text-sm font-medium rounded-md hover:bg-accent hover:text-accent-foreground transition-colors"
              @click="mobileMenuOpen = false"
            >
              {{ button.name }}
            </a>
          </nav>
        </div>
      </header>

      <!-- Main Content -->
      <main :class="['relative', isAdminList && 'md:flex md:min-h-0 md:flex-1 md:flex-col']">
        <router-view @showTooltip="showTooltip" :announcements="announcements" />
      </main>

      <!-- Footer -->
      <footer :class="['border-t mt-auto', isAdminList && 'md:hidden']">
        <div class="container mx-auto px-4 py-6 max-w-7xl">
          <div class="flex flex-col items-center gap-4">
            <div class="text-sm text-muted-foreground text-center">
              Powered by <a href="https://gatus.io" target="_blank" class="font-medium text-emerald-800 hover:text-emerald-600">Gatus</a>
            </div>
            <Social />
          </div>
        </div>
      </footer>
    </div>

    <!-- OIDC Login Screen -->
    <div v-else id="login-container" class="flex items-center justify-center min-h-screen p-4">
      <Card class="w-full max-w-md">
        <CardHeader class="text-center">
          <div v-if="logo" class="flex items-center justify-center gap-4 mb-4">
            <img :src="logo" alt="" class="w-20 h-20 object-contain" />
            <div class="w-px h-12 bg-border"></div>
            <img src="./assets/logo.svg" alt="Gatus" class="w-20 h-20" />
          </div>
          <img v-else src="./assets/logo.svg" alt="Gatus" class="w-20 h-20 mx-auto mb-4" />
          <CardTitle class="text-3xl">{{ header }}</CardTitle>
          <p class="text-muted-foreground mt-2">{{ loginSubtitle }}</p>
        </CardHeader>
        <CardContent>
          <div v-if="route && route.query.error" class="mb-6">
            <div class="p-3 rounded-md bg-destructive/10 border border-destructive/20">
              <p class="text-sm text-destructive text-center">
                <span v-if="route.query.error === 'access_denied'">
                  You do not have access to this status page
                </span>
                <span v-else>{{ route.query.error }}</span>
              </p>
            </div>
          </div>

          <a
            :href="`/oidc/login`"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground hover:bg-primary/90 h-11 px-8 w-full"
            @click="isOidcLoading = true"
          >
            <Loading v-if="isOidcLoading" size="xs" />
            <template v-else>
              <LogIn class="mr-2 h-4 w-4" />
              Login with OIDC
            </template>
          </a>
        </CardContent>
      </Card>
    </div>

    <!-- Tooltip -->
    <Tooltip v-if="routerReady && !isPublic" :result="tooltip.result" :event="tooltip.event" :isPersistent="tooltipIsPersistent" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Menu, X, LogIn } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import Social from './components/Social.vue'
import Tooltip from './components/Tooltip.vue'
import Loading from './components/Loading.vue'
import PublicLayout from './components/public/PublicLayout.vue'

const route = useRoute()
const router = useRouter()

// The layout depends on the route, so nothing is shown before the router resolves the first one
const routerReady = ref(false)
const isPublic = computed(() => route.meta.public === true)
// Administration (fork): compact header on every page, and a layout that fills the window on the lists
const isAdmin = computed(() => route.meta.admin === true)
const isAdminList = computed(() => route.meta.adminList === true)
let configLoadingStarted = false

// State
const retrievedConfig = ref(false)
const config = ref({ oidc: false, authenticated: true })
const announcements = ref([])
const tooltip = ref({})
const mobileMenuOpen = ref(false)
const isOidcLoading = ref(false)
const tooltipIsPersistent = ref(false)
let configInterval = null

// Computed properties
const logo = computed(() => {
  return window.config && window.config.logo && window.config.logo !== '{{ .UI.Logo }}' ? window.config.logo : ""
})

const header = computed(() => {
  return window.config && window.config.header && window.config.header !== '{{ .UI.Header }}' ? window.config.header : "Gatus"
})

const link = computed(() => {
  return window.config && window.config.link && window.config.link !== '{{ .UI.Link }}' ? window.config.link : null
})

const buttons = computed(() => {
  return window.config && window.config.buttons ? window.config.buttons : []
})

const showAdminLink = computed(() => {
  return Boolean(config.value && config.value.admin && config.value.admin.enabled && config.value.admin.authorized)
})

const loginSubtitle = computed(() => {
  return window.config && window.config.loginSubtitle && window.config.loginSubtitle !== '{{ .UI.LoginSubtitle }}' ? window.config.loginSubtitle : "System Monitoring Dashboard"
})

// Methods
const fetchConfig = async () => {
  try {
    const response = await fetch(`/api/v1/config`, { credentials: 'include' })
    if (response.status === 200) {
      const data = await response.json()
      config.value = data
      announcements.value = data.announcements || []
    }
    retrievedConfig.value = true
  } catch (error) {
    console.error('Failed to fetch config:', error)
    retrievedConfig.value = true
  }
}

const showTooltip = (result, event, action = 'hover') => {
  if (action === 'click') {
    if (!result) {
      // Deselecting
      tooltip.value = {}
      tooltipIsPersistent.value = false
    } else {
      // Selecting new data point
      tooltip.value = { result, event }
      tooltipIsPersistent.value = true
    }
  } else if (action === 'hover') {
    // Only update tooltip on hover if not in persistent mode
    if (!tooltipIsPersistent.value) {
      tooltip.value = { result, event }
    }
  }
}

const handleDocumentClick = (event) => {
  // Close persistent tooltip when clicking outside
  if (tooltipIsPersistent.value) {
    const tooltipElement = document.getElementById('tooltip')
    // Check if click is on a data point bar or inside tooltip
    const clickedDataPoint = event.target.closest('.flex-1.h-6, .flex-1.h-8')

    if (tooltipElement && !tooltipElement.contains(event.target) && !clickedDataPoint) {
      tooltip.value = {}
      tooltipIsPersistent.value = false
      // Emit event to clear selections in child components
      window.dispatchEvent(new CustomEvent('clear-data-point-selection'))
    }
  }
}

// The config (and the OIDC login screen) is only needed outside of the public status pages: it is fetched when the first
// non-public route is shown, and only then refreshed every 10 minutes for announcements
watch([routerReady, isPublic], ([ready, isPublicRoute]) => {
  if (!ready || isPublicRoute || configLoadingStarted) {
    return
  }
  configLoadingStarted = true
  fetchConfig()
  configInterval = setInterval(fetchConfig, 600000)
})

onMounted(() => {
  router.isReady().then(() => {
    routerReady.value = true
  })
  // Add click listener for closing persistent tooltips
  document.addEventListener('click', handleDocumentClick)
})

// Clean up interval on unmount
onUnmounted(() => {
  if (configInterval) {
    clearInterval(configInterval)
    configInterval = null
  }
  // Remove click listener
  document.removeEventListener('click', handleDocumentClick)
})
</script>
