import type { Config } from 'tailwindcss';

export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	darkMode: ['class'],

	theme: {
		extend: {
			colors: {
				carbonblue: {
					50: '#ecf5ff',
					100: '#d0e2ff',
					200: '#a6c8ff',
					300: '#77a9fe',
					400: '#4589ff',
					500: '#0e61fe',
					600: '#0043ce',
					700: '#012d9c',
					800: '#001d6c',
					900: '#001141'
				},
				carbongray: {
					50: '#f4f4f4',
					100: '#e0e0e0',
					200: '#c6c6c6',
					300: '#a8a8a8',
					400: '#8d8d8d',
					500: '#6f6f6f',
					600: '#525252',
					700: '#262626',
					800: '#161616',
					900: '#000000'
				},
				carbonborder: {
					300: '#393939',
					200: '#525252',
					100: '#6f6f6f'
				}
			}
		}
	},

	plugins: [require('@tailwindcss/typography')]
} as Config;
