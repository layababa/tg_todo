<script setup lang="ts">
import { computed, ref } from 'vue'

const tabs = ['未完成', '已完成'] as const
const currentTab = ref<typeof tabs[number]>('未完成')

const mockTodos = ref([
  { id: '1', title: 'Telegram To-Do Mini App 样例任务', owner: 'Product', createdAt: 'Just now', done: false }
])

const visibleTodos = computed(() => mockTodos.value.filter(todo => todo.done === (currentTab.value === '已完成')))

const switchTab = (tab: typeof tabs[number]) => {
  currentTab.value = tab
}
</script>

<template>
  <section class="space-y-4">
    <div class="flex gap-3">
      <button
        v-for="tab in tabs"
        :key="tab"
        class="btn btn-sm"
        :class="currentTab === tab ? 'btn-primary' : 'btn-outline'"
        @click="switchTab(tab)"
      >
        {{ tab }}
      </button>
    </div>

    <p class="text-sm opacity-70">Telegram To-Do Mini App 列表视图（示例数据）</p>

    <div v-if="visibleTodos.length === 0" class="alert">
      当前 Tab 没有任务，稍后再来。
    </div>

    <div v-for="todo in visibleTodos" :key="todo.id" class="card bg-base-200 shadow-sm">
      <div class="card-body">
        <h2 class="card-title">{{ todo.title }}</h2>
        <p class="text-sm">创建人：{{ todo.owner }} · {{ todo.createdAt }}</p>
      </div>
    </div>
  </section>
</template>
