import { createRouter, createWebHistory } from "vue-router";

const OnboardingPage = () => import("@/pages/OnboardingPage.vue");
const HomePage = () => import("@/pages/HomePage.vue");
const GroupBindPage = () => import("@/pages/GroupBindPage.vue");

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      redirect: "/onboarding",
    },
    {
      path: "/onboarding",
      name: "onboarding",
      component: OnboardingPage,
    },
    {
      path: "/home",
      name: "home",
      component: HomePage,
    },
    {
      path: "/bind-group/:groupID",
      name: "bind-group",
      component: GroupBindPage,
    },
    {
      path: "/tasks/:id",
      name: "task-detail",
      component: () => import("@/pages/TaskDetailPage.vue"),
    },
    // 增加一个中转路由，处理从 Telegram 直接链接进入但没有 ID 的情况
    {
      path: "/tasks/",
      redirect: "/",
    },
    {
      path: "/settings",
      name: "settings",
      component: () => import("@/pages/SettingsPage.vue"),
    },
    {
      path: "/groups",
      name: "group-list",
      component: () => import("@/pages/GroupListPage.vue"),
    },
  ],
});

// Deep Linking Handler
router.beforeEach((to, from, next) => {
  // Only check on first load (from.name is null/undefined)
  if (from.name) {
    next();
    return;
  }

  // 1. Check Telegram initData (Standard Deep Link)
  const tg = window.Telegram?.WebApp;
  let startParam = tg?.initDataUnsafe?.start_param;

  // 2. Fallback: Check URL Query Params (For WebApp Buttons or Direct URL access)
  // Telegram might pass it as tgWebAppStartParam or we might use our own param
  if (!startParam) {
    const urlParams = new URLSearchParams(window.location.search);
    startParam =
      urlParams.get("start_param") ||
      urlParams.get("tgWebAppStartParam") ||
      urlParams.get("startapp");
  }

  if (startParam && typeof startParam === "string") {
    console.log("[Router] Found start_param:", startParam);

    if (startParam.startsWith("task_")) {
      const taskId = startParam.replace("task_", "");
      // Prevent infinite loop
      if (to.name === "task-detail" && to.params.id === taskId) {
        next();
        return;
      }
      next({ name: "task-detail", params: { id: taskId } });
      return;
    }

    if (startParam.startsWith("bind_")) {
      const groupId = startParam.replace("bind_", "");
      if (to.name === "bind-group" && to.params.groupID === groupId) {
        next();
        return;
      }
      next({ name: "bind-group", params: { groupID: groupId } });
      return;
    }

    if (startParam === "settings") {
      if (to.name === "settings") {
        next();
        return;
      }
      next({ name: "settings" });
      return;
    }
  }

  next();
});

// Chunk Load Error Handler
router.onError((error) => {
  const pattern =
    /Loading chunk (\d)+ failed|Loading CSS chunk (\d)+ failed|Failed to fetch dynamically imported module/;
  const isChunkLoadFailed = error.message.match(pattern);
  const targetPath = router.currentRoute.value.fullPath;

  if (isChunkLoadFailed) {
    console.log("[Router] Chunk load failed, reloading...", error);
    if (!targetPath.includes("reload=true")) {
      // Prevent infinite reload loop if server is down
      // Simple strategy: reload once.
      // For Mini App, simple location.reload() is usually safe enough
      // as state is mostly in URL or server.
      window.location.reload();
    }
  }
});

export default router;
