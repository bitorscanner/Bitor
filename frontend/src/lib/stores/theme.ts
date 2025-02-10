import { writable } from "svelte/store";

// Initialize dark mode based on system preference and localStorage
function createThemeStore() {
  // Create the store with an initial value (we'll set it properly in the browser)
  const { subscribe, set, update } = writable(false);

  return {
    subscribe,
    initialize: () => {
      // Only run in browser
      if (typeof window === "undefined") return;

      // Check localStorage first
      const storedTheme = localStorage.getItem("theme");
      if (storedTheme) {
        const isDark = storedTheme === "dark";
        set(isDark);
        document.documentElement.classList.toggle("dark", isDark);
        return;
      }

      // If no stored preference, check system preference
      const prefersDark = window.matchMedia(
        "(prefers-color-scheme: dark)",
      ).matches;
      set(prefersDark);
      document.documentElement.classList.toggle("dark", prefersDark);
      localStorage.setItem("theme", prefersDark ? "dark" : "light");
    },
    toggle: () => {
      update((isDark) => {
        const newValue = !isDark;
        document.documentElement.classList.toggle("dark", newValue);
        localStorage.setItem("theme", newValue ? "dark" : "light");
        return newValue;
      });
    },
  };
}

export const theme = createThemeStore();
