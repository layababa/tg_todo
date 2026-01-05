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

  // === Safe Area Handling ===
  // Add 'safe-area-ready' class ONLY when Telegram provides valid values (>0).
  // This allows CSS to use a hardcoded fallback (e.g. 32px) by default.
  // === Safe Area Handling ===
  // Add 'safe-area-ready' class ONLY when Telegram provides valid values (>0).
  // This allows CSS to use a hardcoded fallback (e.g. 32px) by default.
  const handleSafeArea = () => {
    const safe = WebApp.safeAreaInset || { top: 0, bottom: 0 };
    const content = WebApp.contentSafeAreaInset || { top: 0, bottom: 0 };

    // Also check CSS variables directly, as WebApp object might lag behind
    const getCssVar = (name: string) => {
      const val = getComputedStyle(document.documentElement)
        .getPropertyValue(name)
        .trim();
      return val.endsWith("px") ? parseFloat(val) : 0;
    };

    const cssSafeTop = getCssVar("--tg-safe-area-inset-top");
    const cssContentTop = getCssVar("--tg-content-safe-area-inset-top");

    const jsTotal = safe.top + content.top;
    const cssTotal = cssSafeTop + cssContentTop;

    console.log("[Main] handleSafeArea check:", {
      js: { safe, content, total: jsTotal },
      css: { safe: cssSafeTop, content: cssContentTop, total: cssTotal },
      hasReadyClass:
        document.documentElement.classList.contains("safe-area-ready"),
    });

    // We consider it "ready" if we have valid values from EITHER JS or CSS
    if (jsTotal > 0 || cssTotal > 0) {
      document.documentElement.classList.add("safe-area-ready");
    }
  };

  // Check immediately and on events
  handleSafeArea();
  // @ts-expect-error event types not yet in sdk
  WebApp.onEvent("safeAreaChanged", handleSafeArea);
  // @ts-expect-error event types not yet in sdk
  WebApp.onEvent("contentSafeAreaChanged", handleSafeArea);

  // Polling fallback: check every 100ms for 3 seconds
  // This ensures we catch the value update even if the event listener fails
  let checks = 0;
  const interval = setInterval(() => {
    handleSafeArea();
    checks++;
    if (checks >= 30) clearInterval(interval);
  }, 100);

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
