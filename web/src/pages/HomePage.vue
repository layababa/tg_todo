<template>
  <main class="relative min-h-screen bg-base-200 px-6 py-10 text-base-content">
    <section class="mx-auto flex max-w-2xl flex-col items-center gap-6 text-center">
      <div class="hero-icon">
        <span class="text-4xl text-primary">
          <i class="ri-dashboard-line" />
        </span>
      </div>
      <h1 class="text-3xl font-light tracking-tight">
        任务主页建设中
      </h1>
      <p class="text-base text-base-content/80">
        Onboarding 已对接后端接口。待任务 API 上线后，会在此页面展示实时任务列表。
      </p>
      <RouterLink class="btn btn-primary" to="/onboarding">
        返回 Onboarding
      </RouterLink>
    </section>
  </main>
</template>

<style scoped>
.hero-icon {
  @apply flex h-24 w-24 items-center justify-center rounded-full border border-dashed border-primary/60 shadow-lg;
}
</style>
