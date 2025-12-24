import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    allowedHosts: ["ddddapp.zcvyzest.xyz"],
  },
  test: {
    globals: true,
    environment: "jsdom",
    coverage: {
      reporter: ["text", "html", "lcov"],
      all: true,
      provider: "v8",
      reportsDirectory: "./coverage",
    },
    setupFiles: "./src/tests/setup.ts",
  },
});
