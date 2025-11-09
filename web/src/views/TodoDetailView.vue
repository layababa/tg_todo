<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const todoId = computed(() => route.params.id ?? '未知')
const title = ref('示例任务标题')
const isDone = ref(false)
</script>

<template>
  <section class="space-y-4">
    <header>
      <p class="text-sm opacity-60">任务 ID：{{ todoId }}</p>
      <h2 class="text-2xl font-semibold mt-1">{{ title }}</h2>
    </header>

    <div class="form-control">
      <label class="label cursor-pointer justify-start gap-3">
        <input v-model="isDone" type="checkbox" class="checkbox checkbox-primary" />
        <span class="label-text">标记为完成</span>
      </label>
    </div>

    <button class="btn btn-primary w-full sm:w-auto">保存更改</button>
  </section>
</template>
