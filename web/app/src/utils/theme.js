// Theme of the screens without the settings of the dashboard (fork): public status pages and login screen. The theme
// cookie is the one of Settings.vue.
const THEME_COOKIE_NAME = 'theme'
const THEME_COOKIE_MAX_AGE = 31536000 // 1 year

export const wantsDarkMode = () => {
  const themeFromCookie = document.cookie.match(new RegExp(`${THEME_COOKIE_NAME}=(dark|light);?`))?.[1]
  return themeFromCookie === 'dark' || (!themeFromCookie && (window.matchMedia('(prefers-color-scheme: dark)').matches || document.documentElement.classList.contains('dark')))
}

// toggleTheme switches between the light and the dark theme, and returns whether the dark theme is now used
export const toggleTheme = () => {
  const theme = wantsDarkMode() ? 'light' : 'dark'
  document.cookie = `${THEME_COOKIE_NAME}=${theme}; path=/; max-age=${THEME_COOKIE_MAX_AGE}; samesite=strict`
  document.documentElement.classList.toggle('dark', theme === 'dark')
  return theme === 'dark'
}
