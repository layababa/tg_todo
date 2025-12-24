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

export default router;
