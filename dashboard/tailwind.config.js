/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'gov': {
          'bg': '#0a0e1a',
          'surface': '#111827',
          'surface-2': '#1a2332',
          'border': '#1e293b',
          'border-light': '#2d3b4e',
          'accent': '#3b82f6',
          'accent-2': '#6366f1',
          'critical': '#ef4444',
          'high': '#f97316',
          'medium': '#eab308',
          'low': '#22c55e',
          'text': '#f1f5f9',
          'text-2': '#94a3b8',
          'text-3': '#64748b',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      animation: {
        'pulse-slow': 'pulse 3s ease-in-out infinite',
        'score-fill': 'scoreFill 1.5s ease-out forwards',
      },
      keyframes: {
        scoreFill: {
          '0%': { strokeDashoffset: '283' },
          '100%': { strokeDashoffset: 'var(--score-offset)' },
        },
      },
    },
  },
  plugins: [],
}
