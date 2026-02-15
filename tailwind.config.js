/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
  ],
  // Ensure primary utilities are always generated (avoids cache/Docker serving old CSS)
  safelist: [
    { pattern: /^(bg|text|border|ring|from|to|via)-primary(-[0-9]+)?$/, variants: ["hover", "focus", "group-hover"] },
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: "#FFF0F7",
          100: "#FFE0F0",
          200: "#FFC2E2",
          300: "#FF99D1",
          400: "#FF66BD",
          500: "#FF369C",
          600: "#E62D8C",
          700: "#CC2080",
          800: "#A3186B",
          900: "#7A1252",
          DEFAULT: "#FF369C",
        },
      },
    },
  },
  plugins: [],
};
