<script setup lang="ts">
import type { Comment } from '@/api/comment'

interface Props {
  comment: Comment
  isReply?: boolean
  parentAuthor?: string
  parentContent?: string
  depth?: number
}

const props = withDefaults(defineProps<Props>(), {
  depth: 0,
  isReply: false
})

const emit = defineEmits(['reply'])

const onReply = (c: Comment) => {
  emit('reply', c)
}

// 资深设计师配色系统：多色阶跨度方案 (Cyberpunk Palette)
const getColorScale = (d: number) => {
  if (!props.isReply) return { bg: 'rgba(255,255,255,0.03)', border: 'rgba(255,255,255,0.1)', rail: 'transparent' }
  
  const scales = [
    { bg: 'rgba(0, 168, 255, 0.08)', border: 'rgba(0, 168, 255, 0.3)', rail: 'rgba(0, 168, 255, 0.5)' }, // Depth 1: Blue
    { bg: 'rgba(157, 0, 255, 0.08)', border: 'rgba(157, 0, 255, 0.3)', rail: 'rgba(157, 0, 255, 0.5)' }, // Depth 2: Purple
    { bg: 'rgba(255, 0, 127, 0.08)', border: 'rgba(255, 0, 127, 0.3)', rail: 'rgba(255, 0, 127, 0.5)' }, // Depth 3: Magenta
    { bg: 'rgba(255, 171, 0, 0.08)', border: 'rgba(255, 171, 0, 0.3)', rail: 'rgba(255, 171, 0, 0.5)' },  // Depth 4: Amber
  ]
  
  return scales[Math.min(d - 1, scales.length - 1)]
}

const theme = getColorScale(props.depth)
</script>

<template>
  <div 
    v-if="comment" 
    class="comment-node flex flex-col w-full relative"
  >
    <!-- Thread Rail removed as requested -->

    <!-- Main Comment Body -->
    <div class="flex gap-2.5 items-start group relative">
      <!-- Avatar - Standardized -->
      <div 
        class="w-8 h-8 rounded-full bg-base-300 flex items-center justify-center font-bold border border-base-content/10 shrink-0 shadow-md"
      >
        {{ (comment.user?.name || 'U').charAt(0).toUpperCase() }}
      </div>

      <!-- Content Area -->
      <div class="flex-1 min-w-0">
        <!-- Header -->
        <div class="flex items-center justify-between mb-1">
          <div class="flex items-center gap-2 overflow-hidden">
            <span class="font-bold text-xs text-base-content/90 truncate">
              {{ comment.user?.name || 'Unknown' }}
            </span>
            <span class="text-[9px] text-base-content/30 font-mono shrink-0">
              {{ new Date(comment.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}
            </span>
          </div>
        </div>

        <!-- Text Bubble with Quote and Color Scale -->
        <div 
          class="text-sm break-words leading-relaxed rounded-lg border transition-all overflow-hidden"
          :style="{ 
            backgroundColor: theme.bg,
            borderColor: theme.border
          }"
        >
          <!-- Quote Section (Context) -->
          <div 
            v-if="isReply && parentContent" 
            class="px-2.5 py-1.5 bg-black/20 border-b border-white/5 flex flex-col gap-0.5"
          >
            <div class="flex items-center gap-1.5 text-[9px] font-bold text-base-content/40 uppercase tracking-widest">
              <i class="ri-chat-quote-line"></i>
              <span>引用 @{{ parentAuthor }}</span>
            </div>
            <div class="text-[11px] text-base-content/40 italic line-clamp-1 border-l border-base-content/20 pl-2">
              {{ parentContent }}
            </div>
          </div>

          <!-- Comment Text -->
          <div class="px-3 py-2 text-base-content/90">
            {{ comment.content }}
          </div>
        </div>

        <!-- Action -->
        <div class="flex items-center mt-1">
          <button 
            @click="onReply(comment)" 
            class="text-[10px] text-base-content/30 hover:text-primary transition-colors font-mono tracking-widest uppercase"
          >
            [ 回复 ]
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.comment-node {
  width: auto;
}
</style>
