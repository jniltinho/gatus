// Theme of the web interface (fork): used by the settings of the dashboard, the public status pages and the login
// screen. The same rule is applied by the inline script of public/index.html before the first paint.
//
// - A valid theme cookie (dark or light) wins.
// - Otherwise the default theme of the server (ui.dark-mode) is used, from the data-default-theme attribute of <html>.
// - The preference of the operating system (prefers-color-scheme) is not used.
const THEME_COOKIE_NAME = 'theme'
const THEME_COOKIE_MAX_AGE = 31536000 // 1 year

export const THEME_COLORS = Object.freeze({ dark: '#030712', light: '#f7f9fb' })

// defaultThemeIsDark returns whether the default theme of the server is dark. Without the attribute, or with the
// template not rendered (development server of Vue), the default is dark
export const defaultThemeIsDark = () => {
  const root = document.documentElement
  const value = root && root.dataset ? root.dataset.defaultTheme : undefined
  if (value === undefined || value === null || value.includes('{{')) {
    return true
  }
  return value === 'dark'
}

// themeFromCookie returns dark or light from the theme cookie, or an empty string when it is absent or invalid
export const themeFromCookie = () => {
  const match = /(?:^|;\s*)theme=([^;]*)/.exec(document.cookie || '')
  const value = match ? match[1].trim() : ''
  return value === 'dark' || value === 'light' ? value : ''
}

export const wantsDarkMode = () => {
  const theme = themeFromCookie()
  return theme ? theme === 'dark' : defaultThemeIsDark()
}

// applyTheme sets the class of <html> and the theme-color of the browser
export const applyTheme = (dark) => {
  document.documentElement.classList.toggle('dark', dark)
  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) {
    meta.setAttribute('content', dark ? THEME_COLORS.dark : THEME_COLORS.light)
  }
}

// toggleTheme switches between the light and the dark theme, and returns whether the dark theme is now used
export const toggleTheme = () => {
  const theme = wantsDarkMode() ? 'light' : 'dark'
  document.cookie = `${THEME_COOKIE_NAME}=${theme}; path=/; max-age=${THEME_COOKIE_MAX_AGE}; samesite=strict`
  applyTheme(theme === 'dark')
  return theme === 'dark'
}
