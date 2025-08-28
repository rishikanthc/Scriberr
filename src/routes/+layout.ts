import { browser } from '$app/environment'
import '$lib/i18n' // Import to initialize. Important :)
import { locale, waitLocale } from 'svelte-i18n'
import type { LayoutLoad } from './$types'

export const prerender = true;

export const load: LayoutLoad = async () => {
	if (browser) {
		const savedLocale = localStorage.getItem('selectedLocale');
		locale.set(savedLocale || window.navigator.language)
	}
	await waitLocale()
}
