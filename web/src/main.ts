import { createApp } from "vue";
import { createPinia } from "pinia";
import WebApp from "@twa-dev/sdk";

import App from "./App.vue";
import router from "./router";
import "./styles/main.css";
import { useAuthStore } from "@/store/auth";

try {
  WebApp.ready();
  WebApp.expand();
  // Disable vertical swipes to prevent "pull-to-collapse" behavior
  // Only available in Telegram WebApp SDK version 7.7 or higher
  const version = parseFloat(WebApp.version);
  if (version >= 7.7 && WebApp.isVerticalSwipesEnabled) {
    WebApp.isVerticalSwipesEnabled = false;
  }

  // === Active Request ===
  // Explicitly request safe area data from Telegram Client
  // This triggers a 'safeAreaChanged' event, ensuring we get data ASAP
  const WebView = (window as any).Telegram?.WebView;
  if (WebView?.postEvent) {
    console.log("[Main] Sending active request for safe area...");
    WebView.postEvent("web_app_request_safe_area");
    WebView.postEvent("web_app_request_content_safe_area");
  }
} catch (err) {
  console.warn("[tg-miniapp] Telegram WebApp bridge missing", err);
}

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);

if (import.meta.env.DEV) {
  const authStore = useAuthStore(pinia);
  // @ts-expect-error expose for debugging
  window.authStore = authStore;
  console.debug("[main] authStore mounted on window.authStore");
}

app.mount("#app");
