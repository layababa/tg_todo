<script setup lang="ts">
import type { Task } from '@/types/todo'

defineProps<{
  task: Task
}>()

defineEmits<{
  toggle: [taskId: string, status: Task['status']]
  select: [taskId: string]
}>()
</script>

<template>
  <article class="card bg-base-200 shadow-sm hover:shadow-md transition cursor-pointer" @click="$emit('select', task.id)">
    <div class="card-body flex gap-4 items-start">
      <label class="relative flex items-center" @click.stop>
        <input
          class="uiverse-checkbox__input"
          type="checkbox"
          :checked="task.status === 'completed'"
          :disabled="!task.permissions.canComplete"
          @change="$emit('toggle', task.id, task.status === 'completed' ? 'pending' : 'completed')"
        />
        <span class="uiverse-checkbox"></span>
      </label>

      <div class="flex-1 space-y-2">
        <div class="flex items-center justify-between gap-3">
          <h3 class="card-title text-base">
            {{ task.title }}
          </h3>
          <span v-if="task.status === 'completed'" class="badge badge-success badge-outline">已完成</span>
        </div>

        <p class="text-sm opacity-70">
          创建人：{{ task.createdBy.displayName }} ·
          {{ new Date(task.createdAt).toLocaleString() }}
        </p>

        <div class="flex flex-wrap gap-2">
          <span v-for="person in task.assignees" :key="person.id" class="badge badge-secondary badge-outline">
            @{{ person.username ?? person.displayName }}
          </span>
        </div>
      </div>
    </div>
  </article>
</template>
