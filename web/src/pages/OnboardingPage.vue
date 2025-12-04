<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useAuthStore } from '@/store/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const startParam = computed(() => (route.query.start_param as string | undefined) ?? undefined)

const isLoading = computed(() => authStore.loadingStatus || authStore.loadingLink)

const statusText = computed(() => {
  if (!authStore.hasInitData) {
    return '缺少 Telegram init data，请从 Telegram Mini App 内打开应用，或调用 window.tgTodo.setMockInitData() 注入测试数据。'
  }
  if (authStore.loadingStatus) {
    return '正在读取您的 Telegram 会话...'
  }
  return authStore.statusMessage
})

onMounted(() => {
  if (authStore.hasInitData) {
    authStore.fetchStatus(startParam.value).catch(() => {
      /* error handled by store */
    })
  }
})

watch(
  () => authStore.notionConnected,
  (connected) => {
    if (connected) {
      router.replace({ name: 'home', query: route.query })
    }
  },
  { immediate: true }
)

watch(
  () => route.query.start_param,
  (param) => {
    if (param && typeof param === 'string') {
      authStore.fetchStatus(param).catch(() => {})
    }
  }
)

const handleConnect = async () => {
  if (!authStore.hasInitData) return
  try {
    const url = await authStore.requestNotionAuthUrl()
    window.location.href = url
  } catch {
    /* error surfaced via store */
  }
}

const handleGuest = async () => {
  if (!authStore.notionConnected && authStore.hasInitData && !authStore.user) {
    await authStore.fetchStatus(startParam.value).catch(() => {})
  }
  router.push({ name: 'home', query: route.query })
}
</script>

<template>
  <main class="relative min-h-screen overflow-hidden bg-[#040404] text-base-content">
    <div class="grid-overlay" />
    <div class="scan-line" />

    <section
      class="relative z-10 mx-auto flex min-h-screen max-w-md flex-col items-center justify-center px-6 text-center"
    >
      <div class="hero-graphic mb-10">
        <i class="ri-link-m" />
      </div>
      <h1 class="text-3xl font-light tracking-tight text-white">系统初始化</h1>
      <p class="mt-3 font-mono text-xs uppercase text-white/60">
        建立 Telegram 与 Notion 的神经连接
      </p>

      <button
        class="btn btn-primary btn-block mt-10"
        :disabled="isLoading || !authStore.hasInitData"
        @click="handleConnect"
      >
        <span v-if="isLoading" class="loading loading-spinner loading-sm mr-2" />
        连接 NOTION 工作区
      </button>

      <p
        v-if="statusText"
        class="mt-6 text-xs font-mono text-white/70"
      >
        {{ statusText }}
      </p>

      <p
        v-if="authStore.error"
        class="mt-2 text-xs font-mono text-error"
      >
        {{ authStore.error }}
      </p>

      <button
        class="btn btn-ghost btn-sm mt-8 font-mono uppercase tracking-widest text-white/80"
        :disabled="authStore.loadingStatus"
        @click="handleGuest"
      >
        [ 游客模式访问 ]
      </button>
    </section>
  </main>
</template>

<style scoped>
.grid-overlay {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(rgba(255, 255, 255, 0.08) 1px, transparent 1px);
  background-size: 20px 20px;
  opacity: 0.35;
}

.scan-line {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 3px;
  background: linear-gradient(90deg, transparent, rgba(171, 246, 0, 0.6), transparent);
  animation: scan 4s linear infinite;
}

.hero-graphic {
  @apply flex h-28 w-28 items-center justify-center rounded-full border border-dashed border-primary/40 shadow-[0_0_30px_rgba(171,246,0,0.3)];
  position: relative;
  color: #abf600;
  font-size: 48px;
}

.hero-graphic::after {
  content: '';
  position: absolute;
  inset: -12px;
  border: 1px solid rgba(171, 246, 0, 0.2);
  border-radius: 9999px;
  animation: orbit 12s linear infinite;
}

@keyframes orbit {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes scan {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(100%);
  }
}
</style>
