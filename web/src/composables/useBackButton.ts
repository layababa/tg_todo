import { onMounted, onUnmounted, watch } from "vue";
import { useRouter, useRoute } from "vue-router";

export function useBackButton() {
  const router = useRouter();
  const route = useRoute();

  let lastBackPressTime = 0;
  const EXIT_THRESHOLD = 2000; // 2 seconds

  const showToast = (msg: string) => {
    // Simple toast implementation
    const toast = document.createElement("div");
    toast.textContent = msg;
    Object.assign(toast.style, {
      position: "fixed",
      bottom: "80px",
      left: "50%",
      transform: "translateX(-50%)",
      backgroundColor: "rgba(0,0,0,0.7)",
      color: "white",
      padding: "10px 20px",
      borderRadius: "20px",
      zIndex: "9999",
      fontSize: "14px",
      backdropFilter: "blur(5px)",
      transition: "opacity 0.3s ease",
    });
    document.body.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = "0";
      setTimeout(() => document.body.removeChild(toast), 300);
    }, 2000);
  };

  const handleBackButton = () => {
    const isRoot = route.name === "home" || route.name === "onboarding";

    if (isRoot) {
      const now = Date.now();
      if (now - lastBackPressTime < EXIT_THRESHOLD) {
        window.Telegram?.WebApp?.close();
      } else {
        lastBackPressTime = now;
        showToast("再按一次退出");
      }
    } else {
      router.back();
    }
  };

  const updateButtonState = () => {
    const tg = window.Telegram?.WebApp;
    if (!tg) return;

    // Always show BackButton to intercept hardware key on Android
    // This is a trade-off: UI shows arrow even on Home, but allows custom exit logic
    if (!tg.BackButton.isVisible) {
      tg.BackButton.show();
    }
  };

  onMounted(() => {
    const tg = window.Telegram?.WebApp;
    if (tg) {
      tg.BackButton.onClick(handleBackButton);
      updateButtonState();
    }
  });

  onUnmounted(() => {
    const tg = window.Telegram?.WebApp;
    if (tg) {
      tg.BackButton.offClick(handleBackButton);
    }
  });

  // Ensure button remains visible on route change
  watch(
    () => route.path,
    () => {
      updateButtonState();
    }
  );
}
