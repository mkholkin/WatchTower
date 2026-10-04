/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        slate: { 100: '#f3f5f6', 200: '#dfe4e7', 300: '#c1c9ce', 400: '#a0aab1', 500: '#8c979f', 600: '#7c8992', 700: '#46515a', 800: '#252c31', 900: '#101a15' },
        up: '#10b981',
        'up-bg': '#022c22',
        down: '#ef4444',
        'down-bg': '#2d0a0a',
        maintenance: '#f59e0b',
        'maintenance-bg': '#2d1a04',
        'app-bg': '#101214',
        'card-bg': '#181b1e',
        'card-hover': '#202428',
        'input-bg': '#101214',
        border: '#2b3035',
        'border-hover': '#3b444b',
        'sidebar-bg': '#141719',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      animation: {
        'fade-in': 'fadeInUp 0.35s ease-out forwards',
        shimmer: 'shimmer 1.5s infinite',
      },
      keyframes: {
        fadeInUp: {
          '0%': { opacity: '0', transform: 'translateY(12px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
      },
    },
  },
  plugins: [],
};
