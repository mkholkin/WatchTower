/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        up: '#10b981',
        'up-bg': '#022c22',
        down: '#ef4444',
        'down-bg': '#2d0a0a',
        maintenance: '#f59e0b',
        'maintenance-bg': '#2d1a04',
        'app-bg': '#0b1120',
        'card-bg': '#111b2b',
        'card-hover': '#152032',
        'input-bg': '#0b1120',
        border: '#1e2a3d',
        'border-hover': '#253349',
        'sidebar-bg': '#0d1525',
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
