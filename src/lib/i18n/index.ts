import { browser } from '$app/environment'
import { init, register } from 'svelte-i18n'

const defaultLocale = 'en'

// Register the locales
register('en', () => import('./locales/en.json'))
register('zh-TW', () => import('./locales/zh-TW.json'))

// Determine initial locale
let initialLocale = defaultLocale;
if (browser) {
  initialLocale = localStorage.getItem('selectedLocale') || window.navigator.language || defaultLocale;
}

init({
  fallbackLocale: defaultLocale,
  initialLocale: initialLocale,
})
