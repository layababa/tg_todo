/**
 * useSafeArea - Global composable for Telegram Mini App Safe Area handling
 *
 * This composable intercepts raw Telegram WebView events to get safe area data
 * directly from event parameters, bypassing the WebApp object which may not sync.
 *
 * Usage:
 *   const { safeAreaTop, safeAreaBottom, isReady } = useSafeArea()
 */
import { ref, readonly } from "vue";

// Global state (shared across all components)
const safeAreaTop = ref(32); // Default 32px fallback
const safeAreaBottom = ref(0);
const isReady = ref(false);

// Raw event data storage
const rawSafeAreaData = ref({ top: 0, bottom: 0, left: 0, right: 0 });
const rawContentSafeAreaData = ref({ top: 0, bottom: 0, left: 0, right: 0 });

// Track if interceptor is already installed
let interceptorInstalled = false;
let originalReceiveEvent: any = null;

const getTelegramWebView = () => (window as any).Telegram?.WebView;

// Update computed values from raw data
const updateSafeAreas = () => {
  const totalTop = rawSafeAreaData.value.top + rawContentSafeAreaData.value.top;
  const totalBottom =
    rawSafeAreaData.value.bottom + rawContentSafeAreaData.value.bottom;

  console.log("[useSafeArea] Updating:", {
    rawSafe: rawSafeAreaData.value,
    rawContent: rawContentSafeAreaData.value,
    calculatedTop: totalTop,
    calculatedBottom: totalBottom,
  });

  if (totalTop > 0) {
    safeAreaTop.value = totalTop;
    isReady.value = true;
  }
  safeAreaBottom.value = totalBottom;
};

// Handler for safe_area_changed event
const handleSafeAreaChanged = (eventData: any) => {
  if (eventData && typeof eventData.top === "number") {
    rawSafeAreaData.value = {
      top: eventData.top || 0,
      bottom: eventData.bottom || 0,
      left: eventData.left || 0,
      right: eventData.right || 0,
    };
    console.log(
      "[useSafeArea] Received safe_area_changed:",
      rawSafeAreaData.value
    );
    updateSafeAreas();
  }
};

// Handler for content_safe_area_changed event
const handleContentSafeAreaChanged = (eventData: any) => {
  if (eventData && typeof eventData.top === "number") {
    rawContentSafeAreaData.value = {
      top: eventData.top || 0,
      bottom: eventData.bottom || 0,
      left: eventData.left || 0,
      right: eventData.right || 0,
    };
    console.log(
      "[useSafeArea] Received content_safe_area_changed:",
      rawContentSafeAreaData.value
    );
    updateSafeAreas();
  }
};

/**
 * Initialize the safe area interceptor (call once in main.ts)
 */
export function initSafeAreaInterceptor() {
  if (interceptorInstalled) {
    console.log("[useSafeArea] Interceptor already installed, skipping");
    return;
  }

  const WebView = getTelegramWebView();
  if (!WebView) {
    console.warn("[useSafeArea] Telegram WebView not available");
    return;
  }

  originalReceiveEvent = WebView.receiveEvent;

  WebView.receiveEvent = function (eventType: string, eventData: any) {
    // Handle safe area events
    if (eventType === "safe_area_changed") {
      handleSafeAreaChanged(eventData);
    } else if (eventType === "content_safe_area_changed") {
      handleContentSafeAreaChanged(eventData);
    }

    // Call original handler
    if (originalReceiveEvent) {
      return originalReceiveEvent.call(this, eventType, eventData);
    }
  };

  interceptorInstalled = true;
  console.log("[useSafeArea] Event interceptor installed");

  // Actively request safe area data
  requestSafeArea();
  setTimeout(requestSafeArea, 100);
  setTimeout(requestSafeArea, 500);
}

/**
 * Request safe area data from Telegram client
 */
export function requestSafeArea() {
  const WebView = getTelegramWebView();
  if (WebView?.postEvent) {
    console.log("[useSafeArea] Requesting safe area data...");
    WebView.postEvent("web_app_request_safe_area");
    WebView.postEvent("web_app_request_content_safe_area");
  }
}

/**
 * Composable hook for components to access safe area values
 */
export function useSafeArea() {
  return {
    safeAreaTop: readonly(safeAreaTop),
    safeAreaBottom: readonly(safeAreaBottom),
    isReady: readonly(isReady),
    // Allow components to request refresh
    requestSafeArea,
  };
}

export default useSafeArea;
