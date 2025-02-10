import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  test: {
    include: ["src/**/*.{test,spec}.{js,ts}"],
  },
  ssr: {
    noExternal: [
      "monaco-editor",
      "@battlefieldduck/xterm-svelte",
      "@blocknote/core",
    ],
  },
  optimizeDeps: {
    include: [
      "monaco-editor",
      "@battlefieldduck/xterm-svelte",
      "@blocknote/core",
    ],
  },
  resolve: {
    alias: {
      "@utils": "/src/routes/utils",
    },
  },
});
