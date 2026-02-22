/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
  ],
  // Ensure primary utilities and hero overlay backgrounds are always generated
  safelist: [
    { pattern: /^(bg|text|border|ring|from|to|via)-primary(-[0-9]+)?$/, variants: ["hover", "focus", "group-hover"] },
    "bg-black/30",
    "bg-black/70",
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
