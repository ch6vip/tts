/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/src/js/**/*.js",
  ],
  theme: {
    extend: {
      colors: {
        'brut-yellow': '#FFE066',
        'brut-cyan': '#67E8F9',
        'brut-pink': '#FCA5A5',
        'brut-green': '#86EFAC',
        'brut-purple': '#C084FC',
      },
      boxShadow: {
        'brut-sm': '3px 3px 0 0 #000',
        'brut': '6px 6px 0 0 #000',
        'brut-lg': '10px 10px 0 0 #000',
        'brut-xl': '14px 14px 0 0 #000',
        'brut-hover': '8px 8px 0 0 #000',
        'brut-active': '3px 3px 0 0 #000',
      },
      fontFamily: {
        'black': ['ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
