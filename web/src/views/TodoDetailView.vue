<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { mainButton } from '@/lib/telegram'
import { useTodoStore } from '@/stores/todo'

const route = useRoute()
const router = useRouter()
const todoStore = useTodoStore()

const taskId = computed(() => String(route.params.id))

const editableTitle = ref('')
const completed = ref(false)

const currentTask = computed(() => todoStore.selectedTask)

const syncLocalState = () => {
  if (!currentTask.value) return
  editableTitle.value = currentTask.value.title
  completed.value = currentTask.value.status === 'completed'
}

const syncMainButton = () => {
  if (!currentTask.value || !currentTask.value.permissions.canEdit) {
    mainButton.hide?.()
    return
  }
  try {
    mainButton.setText?.('保存修改')
    mainButton.show?.()
  } catch {
    // ignore if not running inside Telegram WebApp
  }
}

const handleSave = async () => {
  if (!currentTask.value) return
  await todoStore.updateTask(currentTask.value.id, { title: editableTitle.value })
}

const handleToggle = async () => {
  if (!currentTask.value) return
  const nextStatus = completed.value ? 'completed' : 'pending'
  await todoStore.toggleStatus(currentTask.value.id, nextStatus)
}

const handleDelete = async () => {
  if (!currentTask.value) return
  await todoStore.deleteTask(currentTask.value.id)
  router.push({ name: 'todos' })
}

onMounted(async () => {
  await todoStore.loadTask(taskId.value)
  syncLocalState()
  syncMainButton()
})

watch(currentTask, () => {
  syncLocalState()
  syncMainButton()
})

</script>

<template>
  <section v-if="currentTask" class="space-y-5">
    <header class="space-y-1">
      <p class="text-xs opacity-60">任务 ID：{{ currentTask.id }}</p>
      <div class="flex flex-col gap-2">
        <label class="text-sm font-medium">标题</label>
        <input
          v-model="editableTitle"
          :disabled="!currentTask.permissions.canEdit"
          type="text"
          class="input input-bordered"
          @blur="handleSave"
        />
      </div>
    </header>

    <div class="form-control">
      <label class="label cursor-pointer justify-start gap-3">
        <input
          v-model="completed"
          :disabled="!currentTask.permissions.canComplete"
          type="checkbox"
          class="checkbox checkbox-primary"
          @change="handleToggle"
        />
        <span class="label-text">标记为完成</span>
      </label>
    </div>

    <div class="space-y-2 bg-base-200 rounded-xl p-4">
      <p class="text-sm">
        创建人：<strong>{{ currentTask.createdBy.displayName }}</strong> ·
        {{ new Date(currentTask.createdAt).toLocaleString() }}
      </p>
      <p class="text-sm">指派给：</p>
      <div class="flex flex-wrap gap-2">
        <span v-for="person in currentTask.assignees" :key="person.id" class="badge badge-outline">
          {{ person.displayName }}
        </span>
      </div>
    </div>

    <div class="flex flex-col gap-3">
      <a
        v-if="currentTask.sourceMessageUrl"
        :href="currentTask.sourceMessageUrl"
        target="_blank"
        rel="noreferrer"
        class="btn btn-outline"
      >
        查看 Telegram 原始消息
      </a>
      <button
        v-if="currentTask.permissions.canDelete"
        class="btn btn-error"
        type="button"
        @click="handleDelete"
      >
        删除任务
      </button>
    </div>
  </section>

  <div v-else class="skeleton h-40 w-full rounded-2xl"></div>
</template>
