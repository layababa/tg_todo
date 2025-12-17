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
    proxy: {
      "/auth": "http://localhost:8080",
      "/groups": "http://localhost:8080",
      "/tasks": "http://localhost:8080",
      "/databases": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
      "/webhook": "http://localhost:8080",
    },
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
