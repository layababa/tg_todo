<script setup lang="ts">
import type { TaskStatus } from '@/types/todo'

const tabs: { label: string; value: TaskStatus }[] = [
  { label: '未完成 Pending', value: 'pending' },
  { label: '已完成 Completed', value: 'completed' }
]

defineProps<{
  modelValue: TaskStatus
}>()

const emit = defineEmits<{
  'update:modelValue': [value: TaskStatus]
}>()

const selectTab = (value: TaskStatus) => {
  emit('update:modelValue', value)
}
</script>

<template>
  <div class="join">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      class="join-item btn btn-sm font-medium"
      :class="modelValue === tab.value ? 'btn-primary' : 'btn-outline'"
      type="button"
      @click="selectTab(tab.value)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>
