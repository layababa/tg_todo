<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'

import StatusTabs from '@/components/todo/StatusTabs.vue'
import TodoCard from '@/components/todo/TodoCard.vue'
import { useTodoStore } from '@/stores/todo'

const todoStore = useTodoStore()
const router = useRouter()

const { activeTab, isLoading, error } = storeToRefs(todoStore)
const tasks = computed(() => todoStore.activeTasks)

const handleToggle = async (taskId: string, status: 'pending' | 'completed') => {
  await todoStore.toggleStatus(taskId, status)
}

const openDetail = (taskId: string) => {
  router.push({ name: 'todo-detail', params: { id: taskId } })
}

onMounted(async () => {
  if (!todoStore.items.length) {
    await todoStore.fetchAll()
  }
})
</script>

<template>
  <section class="space-y-4">
    <header class="space-y-2">
      <div class="flex items-center justify-between gap-3 flex-wrap">
        <h2 class="text-xl font-semibold">待办任务</h2>
        <StatusTabs v-model="activeTab" />
      </div>
      <p class="text-sm opacity-80">数据示例来源：PRD 中的 Pending / Completed 任务流程。</p>
    </header>

    <div v-if="isLoading" class="skeleton h-24 w-full rounded-xl"></div>
    <p v-else-if="error" class="alert alert-error">{{ error }}</p>
    <p v-else-if="tasks.length === 0" class="alert alert-info">当前 Tab 没有任务，稍后再来。</p>

    <div v-else class="space-y-3">
      <TodoCard
        v-for="task in tasks"
        :key="task.id"
        :task="task"
        @toggle="handleToggle"
        @select="openDetail"
      />
    </div>
  </section>
</template>
