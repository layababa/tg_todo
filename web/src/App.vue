<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <div class="min-h-screen bg-base-100 text-base-content">
    <header class="px-4 py-3 border-b border-base-200 flex items-center justify-between">
      <div>
        <p class="text-sm opacity-70 uppercase tracking-wide">Telegram To-Do</p>
        <h1 class="text-xl font-semibold">Mini App 控制面板</h1>
      </div>
      <span class="badge badge-secondary">Alpha</span>
    </header>

    <main class="p-4">
      <RouterView />
    </main>
  </div>
</template>
