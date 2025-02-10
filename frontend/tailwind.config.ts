const config = {
  content: [
    "./src/**/*.{html,js,svelte,ts}",
    "./node_modules/flowbite-svelte/**/*.{html,js,svelte,ts}",
    "./node_modules/flowbite-svelte-blocks/**/*.{html,js,svelte,ts}",
  ],

  plugins: [require("flowbite/plugin"), require("flowbite-typography")],

  darkMode: "class",

  theme: {
    extend: {
      colors: {
        // flowbite-svelte
        primary: {
          50: "#f0fff4",
          100: "#dcfff0",
          200: "#b4ffda",
          300: "#72ffb6",
          400: "#33ff88",
          500: "#00ff21", // Main Matrix green
          600: "#00e639",
          700: "#00cc33",
          800: "#00b32d",
          900: "#009424",
        },
        secondary: {
          50: "#f0f9ff",
          100: "#e0f2fe",
          200: "#bae6fd",
          300: "#7dd3fc",
          400: "#38bdf8",
          500: "#0ea5e9",
          600: "#0284c7",
          700: "#0369a1",
          800: "#075985",
          900: "#0c4a6e",
        },
        accent: {
          50: "#fdf4ff",
          100: "#fae8ff",
          200: "#f5d0fe",
          300: "#f0abfc",
          400: "#e879f9",
          500: "#d946ef",
          600: "#c026d3",
          700: "#a21caf",
          800: "#86198f",
          900: "#701a75",
        },
      },
    },
  },
};

module.exports = config;
