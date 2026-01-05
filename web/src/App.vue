<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useBackButton } from '@/composables/useBackButton'

// Initialize Global Back Button Handling (for Android hardware key support)
useBackButton()

const route = useRoute()
const transitionName = ref('page-slide-right')

// 监听路由深度或历史记录来决定动画方向
// 监听路由深度或历史记录来决定动画方向
watch(() => route.path, (toPath, fromPath) => {
  // Disable transition for Onboarding page
  if (toPath === '/onboarding' || toPath === '/') {
    transitionName.value = ''
    return
  }

  const toDepth = toPath.split('/').filter(Boolean).length
  const fromDepth = fromPath ? fromPath.split('/').filter(Boolean).length : 0
  
  if (toDepth < fromDepth || toPath === '/home') {
    transitionName.value = 'page-slide-right' // Back/Pop
  } else {
    transitionName.value = 'page-slide-left' // Forward/Push
  }
})
</script>

<template>
  <!-- Global Background (Static) -->
  <div class="grid-bg"></div>
  <div class="scan-line"></div>

  <router-view v-slot="{ Component }">
    <transition :name="transitionName">
      <component :is="Component" class="absolute inset-0 w-full h-full bg-black will-change-transform" />
    </transition>
  </router-view>
</template>

<style>
/* iOS-style Slide Transitions */
/* Slide Left (Push: Enter from Right, Exit to Left) */
.page-slide-left-enter-active,
.page-slide-left-leave-active,
.page-slide-right-enter-active,
.page-slide-right-leave-active {
  transition: transform 0.4s cubic-bezier(0.165, 0.84, 0.44, 1), opacity 0.4s ease;
  will-change: transform;
}

/* PUSH: New page enters from right */
.page-slide-left-enter-from {
  transform: translate3d(100%, 0, 0);
  z-index: 10;
}
.page-slide-left-enter-to {
  transform: translate3d(0, 0, 0);
  z-index: 10;
}

/* PUSH: Old page slides slightly left (Parallax) and dims */
.page-slide-left-leave-from {
  transform: translate3d(0, 0, 0);
  z-index: 1;
  filter: brightness(1);
}
.page-slide-left-leave-to {
  transform: translate3d(-30%, 0, 0);
  z-index: 1;
  filter: brightness(0.5);
}

/* POP: Old page (current) slides out to right */
.page-slide-right-leave-from {
  transform: translate3d(0, 0, 0);
  z-index: 10;
}
.page-slide-right-leave-to {
  transform: translate3d(100%, 0, 0);
  z-index: 10;
}

/* POP: New page (previous) slides in from left parallax */
.page-slide-right-enter-from {
  transform: translate3d(-30%, 0, 0);
  z-index: 1;
  filter: brightness(0.5);
}
.page-slide-right-enter-to {
  transform: translate3d(0, 0, 0);
  z-index: 1;
  filter: brightness(1);
}
</style>
