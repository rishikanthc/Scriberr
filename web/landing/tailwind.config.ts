import type { Config } from 'tailwindcss';

export default {
  content: [
    './index.html',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        accent: '#6b8afd',
        accent2: '#a78bfa'
      },
      borderRadius: {
        '2xl': '1rem',
        '3xl': '1.5rem'
      },
      fontFamily: {
        sans: [
          'DM Sans',
          'Inter',
          'ui-sans-serif',
          'system-ui',
          'sans-serif',
        ],
        display: [
          'Outfit',
          'DM Sans',
          'sans-serif',
        ],
        accent: [
          'Outfit',
          'DM Sans',
          'sans-serif',
        ]
      },
      boxShadow: {
        subtle: '0 1px 0 rgba(17,24,39,0.04) inset',
        card: '0 1px 2px rgba(16,24,40,0.06), 0 1px 3px rgba(16,24,40,0.04)',
        soft: '0 2px 10px rgba(0,0,0,0.04)',
        hover: '0 6px 24px rgba(0,0,0,0.06)'
      },
      backgroundImage: {
        'hero-blur': 'radial-gradient(600px 300px at 70% 20%, rgba(107,138,253,0.18), transparent 40%), radial-gradient(700px 300px at 20% 10%, rgba(167,139,250,0.15), transparent 45%)',
        grid: 'linear-gradient(to right, rgba(17,24,39,0.04) 1px, transparent 1px), linear-gradient(to bottom, rgba(17,24,39,0.04) 1px, transparent 1px)'
      },
      backgroundSize: {
        grid: '40px 40px'
      }
    }
  },
  darkMode: 'class'
} satisfies Config;
