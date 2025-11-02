/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./internal/server/static/**/*.{html,js}",
    "./internal/auth/static/**/*.html"
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
