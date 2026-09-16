import { afterEach, test } from 'node:test'
import assert from 'node:assert/strict'
import { defaultThemeIsDark, toggleTheme, wantsDarkMode } from './theme.js'

// A fake document with the data-default-theme attribute (undefined when absent), the cookie and the theme-color meta
const setDocument = ({ defaultTheme, cookie = '' } = {}) => {
  const classes = new Set()
  const meta = { content: '#f7f9fb', setAttribute(name, value) { this[name] = value } }
  const fake = {
    cookie,
    documentElement: {
      dataset: defaultTheme === undefined ? {} : { defaultTheme },
      classList: { toggle: (name, force) => (force ? classes.add(name) : classes.delete(name)) }
    },
    querySelector: (selector) => (selector === 'meta[name="theme-color"]' ? meta : null)
  }
  global.document = fake
  return { fake, classes, meta }
}

afterEach(() => {
  delete global.document
})

test('the default theme comes from data-default-theme', () => {
  setDocument({ defaultTheme: 'dark' })
  assert.equal(defaultThemeIsDark(), true)
  assert.equal(wantsDarkMode(), true)
  setDocument({ defaultTheme: '' })
  assert.equal(defaultThemeIsDark(), false)
  assert.equal(wantsDarkMode(), false)
})

test('without the attribute or with the template not rendered, the default is dark', () => {
  setDocument()
  assert.equal(defaultThemeIsDark(), true)
  setDocument({ defaultTheme: '{{ .DefaultTheme }}' })
  assert.equal(defaultThemeIsDark(), true)
  assert.equal(wantsDarkMode(), true)
})

test('a valid cookie wins over the default theme', () => {
  setDocument({ defaultTheme: 'dark', cookie: 'other=1; theme=light' })
  assert.equal(wantsDarkMode(), false)
  setDocument({ defaultTheme: '', cookie: 'theme=dark; other=1' })
  assert.equal(wantsDarkMode(), true)
})

test('an invalid cookie is ignored', () => {
  setDocument({ defaultTheme: '', cookie: 'theme=blue' })
  assert.equal(wantsDarkMode(), false)
  setDocument({ defaultTheme: 'dark', cookie: 'theme=' })
  assert.equal(wantsDarkMode(), true)
  setDocument({ defaultTheme: 'dark', cookie: 'mytheme=light' })
  assert.equal(wantsDarkMode(), true)
})

test('toggling saves the cookie and updates the class and the theme-color', () => {
  const { fake, classes, meta } = setDocument({ defaultTheme: 'dark' })
  assert.equal(toggleTheme(), false)
  assert.match(fake.cookie, /^theme=light; path=\/;/)
  assert.equal(classes.has('dark'), false)
  assert.equal(meta.content, '#f7f9fb')
  fake.cookie = 'theme=light'
  assert.equal(toggleTheme(), true)
  assert.equal(classes.has('dark'), true)
  assert.equal(meta.content, '#030712')
})
