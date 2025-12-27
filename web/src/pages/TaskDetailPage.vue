<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getTask, patchTask, createTask, deleteTask } from '@/api/task'
import { showToast } from '@/utils/toast'
import { listComments, createComment, type Comment } from '@/api/comment'
import CommentItem from '@/components/CommentItem.vue'
import { useAuthStore } from '@/store/auth'
import { useSwipeBack } from '@/composables/useSwipeBack'
import type { Task } from '@/types/task'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// Add swipe back support
useSwipeBack()

// State
const task = ref<Task | null>(null)
const comments = ref<Comment[]>([])
const loading = ref(true)
const errorMessage = ref('')
const contextOpen = ref(false)

const newComment = ref('')
const isNew = computed(() => route.params.id === 'new')
const isSaving = ref(false)
const replyToCommentID = ref<string | null>(null)
const replyToUserName = ref<string>('')

// Due Date State
const showDatePicker = ref(false)
const dueAtLocal = ref('')

const assignTask = () => {
    if (!task.value?.ID) return
    if (!window.Telegram?.WebApp) {
        console.warn('Telegram WebApp is not available')
        return
    }
    // Deep link format: assign <TaskID>
    // This will open chat selection, then insert "@CheckMyTodoBot assign <TaskID>" into input
    window.Telegram.WebApp.switchInlineQuery(`assign ${task.value.ID}`, ['users', 'groups', 'channels'])
}

const toggleDatePicker = () => {
    if (task.value?.DueAt) {
        const d = new Date(task.value.DueAt)
        const pad = (n: number) => n.toString().padStart(2, '0')
        dueAtLocal.value = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
    } else {
        dueAtLocal.value = ''
    }
    showDatePicker.value = true
}

const updateDueDate = async () => {
    if (!task.value || !task.value.ID) return
    const newDate = dueAtLocal.value ? new Date(dueAtLocal.value).toISOString() : null
    const originalDate = task.value.DueAt
    task.value.DueAt = newDate as any
    try {
        await patchTask(task.value.ID, { due_at: newDate as any })
        showDatePicker.value = false
    } catch (e: any) {
        task.value.DueAt = originalDate
        showToast('更新失败', 'error')
    }
}

const clearDueDate = async () => {
    if (!task.value || !task.value.ID) return
    const originalDate = task.value.DueAt
    task.value.DueAt = null as any
    try {
        await patchTask(task.value.ID, { due_at: null as any })
        showDatePicker.value = false
    } catch (e: any) {
        task.value.DueAt = originalDate
        showToast('清除失败', 'error')
    }
}

const formatDueDate = (dateStr: string | null | undefined) => {
    if (!dateStr) return '设置截止日期'
    const d = new Date(dateStr)
    return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

const contextItems = computed(() => {
    if (!task.value?.Snapshots) return []
    return task.value.Snapshots.map(s => ({
        name: s.Role === 'me' ? 'Me' : (s.Author || 'User'),
        text: s.Text,
        isMe: s.Role === 'me'
    }))
})

const organizedComments = computed(() => {
    if (!comments.value.length) return []
    const map = new Map<string, any>()
    const rootComments: any[] = []
    comments.value.forEach(c => map.set(c.id, { ...c, children: [] }))
    const sortedRaw = [...comments.value].sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
    sortedRaw.forEach(c => {
        const comment = map.get(c.id)
        if (c.parent_id && map.has(c.parent_id)) {
            map.get(c.parent_id).children.push(comment)
        } else {
            rootComments.push(comment)
        }
    })
    const flatten = (comment: any, targetList: any[], depth: number) => {
        comment.children.sort((a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
        comment.children.forEach((child: any) => {
            child._replyToAuthor = comment.user?.name || 'Unknown'
            child._parentContent = comment.content
            child._depth = depth
            targetList.push(child)
            flatten(child, targetList, depth + 1)
        })
    }
    const finalThreads = rootComments.map(root => {
        const allReplies: any[] = []
        flatten(root, allReplies, 1)
        return { ...root, allReplies }
    })
    return finalThreads.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
})

const isReadyForHeavyContent = ref(false)
const isEditingDescription = ref(false)

const resizeTextarea = (el: HTMLTextAreaElement) => {
    el.style.height = 'auto'
    el.style.height = el.scrollHeight + 'px'
}

const startEditingDescription = () => {
    isEditingDescription.value = true
    nextTick(() => {
        const textarea = document.querySelector('.description-textarea') as HTMLTextAreaElement
        if (textarea) {
            resizeTextarea(textarea)
            textarea.focus()
        }
    })
}

const finishEditingDescription = () => {
    if (task.value?.Description) {
        isEditingDescription.value = false
    }
    saveTask()
}

const fetchData = async () => {
    if (isNew.value) {
        task.value = {
            ID: '', Title: '', Status: 'To Do', SyncStatus: 'Pending',
            CreatedAt: new Date().toISOString(), UpdatedAt: new Date().toISOString(),
            Snapshots: [], Assignees: []
        } as Task
        loading.value = false
        isReadyForHeavyContent.value = true
        nextTick(() => document.querySelector('textarea')?.focus())
        return
    }
    
    // 1. Fetch Task immediately for TTI (Time to Interactive)
    const id = route.params.id as string
    try {
        const t = await getTask(id)
        task.value = t
        loading.value = false // Task renders, transition continues smoothly
    } catch (e) {
        errorMessage.value = 'Failed to load task'
        loading.value = false
        return
    }

    // 2. Fetch Comments separately
    try {
        const c = await listComments(id)
        comments.value = c || []
    } catch (e) {
        console.error('Failed to load comments')
    }

    // 3. Enable heavy rendering after transition (approx 350ms)
    setTimeout(() => {
        isReadyForHeavyContent.value = true
    }, 400)
}

const saveTask = async () => {
    if (!task.value || !task.value.Title.trim() || isSaving.value) return
    isSaving.value = true
    try {
        if (isNew.value) {
            const created = await createTask({ title: task.value.Title, description: task.value.Description })
            router.replace(`/tasks/${created.ID}`)
            task.value = created
        } else {
             await patchTask(task.value.ID, { title: task.value.Title, description: task.value.Description })
        }
    } catch (e) {
        console.error('Failed to save task', e)
    } finally {
        isSaving.value = false
    }
}

const updateTitle = async () => {
    if (isNew.value) return
    if (task.value?.Title) saveTask()
}

const cycleStatus = async () => {
    if (!task.value || !task.value.ID) return
    const statusMap: Record<string, string> = { 'To Do': 'In Progress', 'In Progress': 'Done', 'Done': 'To Do' }
    const nextStatus = statusMap[task.value.Status || 'To Do'] || 'To Do'
    const originalStatus = task.value.Status
    task.value.Status = nextStatus as any
    try {
        await patchTask(task.value.ID, { status: nextStatus as any })
    } catch (e: any) {
        task.value.Status = originalStatus
        if (e.response?.status === 403) showToast('权限不足', 'error')
    }
}

const submitComment = async () => {
    if (!newComment.value.trim() || !task.value?.ID) return
    const content = newComment.value
    const parentID = replyToCommentID.value
    newComment.value = ''; replyToCommentID.value = null; replyToUserName.value = ''
    try {
        const comment = await createComment(task.value.ID, { content, parent_id: parentID || undefined })
        if (!comment.user && authStore.user) {
            comment.user = { id: authStore.user.id, name: authStore.user.name, photo_url: authStore.user.photo_url }
        }
        comments.value.push(comment)
        nextTick(() => {
            const el = document.querySelector('footer textarea') as HTMLTextAreaElement
            if (el) el.style.height = 'auto'
        })
    } catch (e) {
        newComment.value = content; replyToCommentID.value = parentID
    }
}

const setReply = (comment: Comment) => {
    replyToCommentID.value = comment.id; replyToUserName.value = comment.user?.name || 'Unknown'
    nextTick(() => {
        const el = document.querySelector('footer textarea') as HTMLTextAreaElement
        el?.focus()
    })
}

const cancelReply = () => { replyToCommentID.value = null; replyToUserName.value = '' }

const onDelete = async () => {
    if (!confirm('Are you sure?')) return
    try {
        if (task.value?.ID) await deleteTask(task.value.ID)
        router.push('/home')
    } catch (e) { alert('Failed') }
}

const toggleContext = () => contextOpen.value = !contextOpen.value
const onFocusComment = () => {} // No-op now, just for event binding if needed
const goBack = () => router.push('/home')

onMounted(fetchData)
</script>

<template>
  <div class="page-root flex flex-col h-full overflow-hidden">
    <div class="grid-bg"></div>
    <div class="scan-line"></div>

    <div class="app-container flex flex-col h-full w-full relative !min-h-0 !pb-0">
        <header class="header shrink-0 z-30 bg-base-100/95 backdrop-blur-md">
            <div class="flex items-center justify-between">
                <button @click="goBack" class="icon-btn tech-btn !w-auto !px-2 !border-none text-sm gap-2">
                    <i class="ri-arrow-left-line"></i> 返回
                </button>
                <div class="flex items-center gap-2">
                    <a v-if="task?.NotionURL" :href="task.NotionURL" target="_blank" class="icon-btn tech-btn !w-auto !px-2 !border-none text-sm gap-2">
                        <i class="ri-notion-fill text-white"></i> Notion
                    </a>
                    <div v-if="!isNew" class="dropdown dropdown-end">
                      <button tabindex="0" class="icon-btn tech-btn !border-none"><i class="ri-more-2-fill"></i></button>
                      <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52 border border-base-content/10">
                          <li><a @click="onDelete" class="text-error"><i class="ri-delete-bin-line"></i> Delete Task</a></li>
                      </ul>
                    </div>
                    <button v-else @click="saveTask" class="btn btn-sm btn-primary" :disabled="isSaving || !task?.Title">
                        {{ isSaving ? 'Saving...' : 'Create' }}
                    </button>
                </div>
            </div>
        </header>

        <main class="flex-1 overflow-y-auto px-4 pt-4 pb-4 scroll-smooth">
            <div v-if="loading" class="flex justify-center mt-20">
                <span class="loading loading-spinner text-primary"></span>
            </div>
            <div v-else-if="task">
                <div class="flex items-center justify-between gap-4 mb-4">
                    <textarea v-model="task.Title" @blur="updateTitle" placeholder="Task Title..." class="detail-title-input flex-1 bg-transparent border-b border-transparent focus:border-primary text-2xl font-bold outline-none resize-none overflow-hidden" rows="1"></textarea>
                    <div class="flex items-center gap-2">
                        <button v-show="task.Title" @click="saveTask" class="btn btn-circle btn-ghost btn-sm text-primary">
                            <i class="ri-check-line text-xl"></i>
                        </button>
                    </div>
                </div>

                <!-- Redesigned Property Grid (Scheme B - 3 Rows) -->
                <div class="grid grid-cols-2 gap-3 mb-8">
                    <!-- 1.1 Status (状态) -->
                    <div @click="cycleStatus" class="bg-base-200/50 p-3 rounded-lg border border-base-content/5 relative overflow-hidden group active:scale-95 transition-all cursor-pointer">
                        <div class="flex items-center gap-2 mb-1 opacity-50">
                            <i class="ri-loader-2-line text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">状态</span>
                        </div>
                        <div class="font-bold text-sm flex items-center gap-2" :class="task?.Status === 'Done' ? 'text-success' : 'text-primary'">
                            {{ {'To Do': '待办', 'In Progress': '进行中', 'Done': '已完成'}[task?.Status || ''] || task?.Status || '待办' }}
                        </div>
                        <div class="absolute right-2 top-2 opacity-0 group-hover:opacity-20 transition-opacity"><i class="ri-arrow-right-line"></i></div>
                    </div>

                    <!-- 1.2 Due Date (截止日期) -->
                    <div @click="toggleDatePicker" class="bg-base-200/50 p-3 rounded-lg border border-base-content/5 relative overflow-hidden group active:scale-95 transition-all cursor-pointer">
                        <div class="flex items-center gap-2 mb-1 opacity-50">
                            <i class="ri-calendar-event-line text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">截止日期</span>
                        </div>
                        <div class="font-bold text-sm" :class="task?.DueAt ? 'text-primary' : 'text-base-content/30'">
                            {{ task?.DueAt ? formatDueDate(task.DueAt) : '未设置' }}
                        </div>
                    </div>

                    <!-- 2.1 Assignee (负责人) -->
                    <div @click="assignTask" class="bg-base-200/50 p-3 rounded-lg border border-base-content/5 relative overflow-hidden group active:scale-95 transition-all cursor-pointer">
                        <div class="flex items-center gap-2 mb-1 opacity-50">
                            <i class="ri-user-3-line text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">负责人</span>
                        </div>
                        <div class="flex items-center gap-2">
                             <div v-if="task?.Assignees && task.Assignees.length > 0" class="flex -space-x-1">
                                <div v-for="user in task.Assignees" :key="user.id" class="w-4 h-4 rounded-full bg-gradient-to-tr from-primary to-secondary flex items-center justify-center text-[8px] text-black font-bold ring-2 ring-base-100/50" :title="user.name">
                                    {{ user.name[0].toUpperCase() }}
                                </div>
                            </div>
                            <div v-else class="w-4 h-4 rounded-full bg-base-content/10 flex items-center justify-center text-[8px] text-base-content/50 font-bold">
                                ?
                            </div>
                            <span class="font-bold text-sm truncate">
                                {{ (task?.Assignees && task.Assignees.length > 0) ? (task.Assignees[0].id === authStore.user?.id ? '我' : task.Assignees[0].name) : '点击指派' }}
                            </span>
                             <i class="ri-share-forward-line text-xs opacity-50 ml-auto"></i>
                        </div>
                    </div>

                    <!-- 2.2 Priority (优先级) - Mocked -->
                    <div class="bg-base-200/50 p-3 rounded-lg border border-base-content/5 relative overflow-hidden opacity-80">
                         <div class="flex items-center gap-2 mb-1 opacity-50">
                            <i class="ri-flag-2-line text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">优先级</span>
                        </div>
                        <div class="font-bold text-sm flex items-center gap-1">
                            <span class="w-2 h-2 rounded-full bg-blue-400"></span>普通
                        </div>
                    </div>

                    <!-- 3.1 Creator (创建人) - Click to PM -->
                    <a v-if="task?.Creator?.tg_username" :href="`https://t.me/${task.Creator.tg_username}`" target="_blank" class="bg-base-200/50 p-3 rounded-lg border border-base-content/5 relative overflow-hidden group active:scale-95 transition-all cursor-pointer">
                        <div class="flex items-center gap-2 mb-1 opacity-50">
                            <i class="ri-user-add-line text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">创建人</span>
                        </div>
                        <div class="font-bold text-sm text-primary flex items-center gap-1 truncate">
                             @{{ task.Creator.tg_username }} <i class="ri-external-link-line text-xs opacity-50"></i>
                        </div>
                    </a>
                    <div v-else class="bg-base-200/50 p-3 rounded-lg border border-base-content/5 relative overflow-hidden opacity-50">
                         <div class="flex items-center gap-2 mb-1 opacity-50">
                            <i class="ri-user-add-line text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">创建人</span>
                        </div>
                        <div class="font-bold text-sm truncate">
                             {{ task?.Creator?.name || '未知' }}
                        </div>
                    </div>

                    <!-- 3.2 Source (来源) - Jump to Context -->
                    <a v-if="task?.ChatJumpURL" :href="task.ChatJumpURL" target="_blank" class="bg-[#24A1DE]/10 p-3 rounded-lg border border-[#24A1DE]/20 relative overflow-hidden group active:scale-95 transition-all cursor-pointer">
                        <div class="flex items-center gap-2 mb-1 text-[#24A1DE]/70">
                            <i class="ri-telegram-fill text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">来源</span>
                        </div>
                        <div class="font-bold text-sm text-[#24A1DE] flex items-center gap-1">
                            Telegram <i class="ri-external-link-line text-xs opacity-50"></i>
                        </div>
                    </a>
                    <a v-else-if="task?.NotionURL" :href="task.NotionURL" target="_blank" class="bg-base-200/50 p-3 rounded-lg border border-base-content/5 relative overflow-hidden group active:scale-95 transition-all cursor-pointer">
                         <div class="flex items-center gap-2 mb-1 opacity-50">
                            <i class="ri-notion-fill text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">来源</span>
                        </div>
                        <div class="font-bold text-sm flex items-center gap-1">
                            Notion <i class="ri-external-link-line text-xs opacity-50"></i>
                        </div>
                    </a>
                    <div v-else class="bg-base-200/20 p-3 rounded-lg border border-base-content/5 opacity-50">
                        <div class="flex items-center gap-2 mb-1">
                            <i class="ri-link text-xs"></i>
                            <span class="text-[10px] uppercase font-mono tracking-wider">来源</span>
                        </div>
                        <div class="text-xs italic">本地任务</div>
                    </div>
                </div>

                <div v-if="contextItems.length > 0" class="context-snapshot-section">
                    <div @click="toggleContext" class="section-toggle cursor-pointer select-none">
                        <div class="flex items-center gap-2 text-sm font-mono"><i class="ri-chat-history-line"></i><span>上下文快照 ({{ contextItems.length }}条)</span></div>
                        <div class="flex items-center gap-2">
                            <!-- Removed redundant 'View Full Record' link as per request -->
                            <i class="ri-arrow-down-s-line transition-transform duration-300" :class="{ 'rotate-180': !contextOpen }"></i>
                        </div>
                    </div>
                    <div v-show="contextOpen" class="context-content">
                        <div v-for="(msg, idx) in contextItems" :key="idx" :class="['chat-bubble', msg.isMe ? 'me' : 'other']">
                            <div v-if="!msg.isMe" class="chat-name">{{ msg.name }}</div>
                            <div class="chat-text">{{ msg.text }}</div>
                        </div>
                    </div>
                </div>

                <!-- Reduced padding to py-2 for slimmer look -->
                <!-- Description Section (Staged Loading) -->
                <div v-if="!isReadyForHeavyContent" class="mt-6 px-4">
                     <div class="h-24 w-full bg-base-content/10 rounded animate-pulse"></div>
                </div>

                <!-- Read Mode: Collapsed View -->
                <div v-if="!isEditingDescription && task.Description" 
                     @click="startEditingDescription"
                     class="editor-content tech-border px-4 py-2 rounded-none mt-6 relative group bg-base-100/10 hover:bg-base-100/20 cursor-pointer min-h-[36px]">
                    <div class="text-sm font-mono leading-relaxed text-base-content/90 line-clamp-2 whitespace-pre-wrap break-all">
                        {{ task.Description }}
                    </div>
                    <!-- Decorators -->
                    <div class="absolute top-0 left-0 w-1.5 h-1.5 border-t border-l border-primary/30"></div>
                    <div class="absolute bottom-0 right-0 w-1.5 h-1.5 border-b border-r border-primary/30"></div>
                    <!-- Expand Hint -->
                    <div class="absolute bottom-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity">
                         <i class="ri-expand-up-down-line text-xs text-primary/40"></i>
                    </div>
                </div>

                <!-- Edit Mode / Empty Mode -->
                <div v-else 
                     class="editor-content tech-border px-4 py-2 rounded-none mt-6 relative group transition-all duration-300 bg-base-100/10 hover:bg-base-100/20" 
                     :class="{ 'min-h-[36px]': !task.Description }">
                    <div v-if="!task.Description" class="absolute inset-0 flex items-center px-4 pointer-events-none">
                        <span class="text-primary/40 font-mono text-sm tracking-wide transform -translate-y-px">
                            <span class="animate-pulse mr-2">›_</span>点击添加任务描述...
                        </span>
                    </div>
                    <textarea 
                        v-model="task.Description" 
                        class="description-textarea w-full bg-transparent border-none outline-none text-sm resize-none text-base-content/80 focus:text-white transition-all font-mono leading-relaxed placeholder-transparent overflow-hidden"
                        :class="{ 'h-[20px]': !task.Description }"
                        @blur="finishEditingDescription"
                        @input="(e) => resizeTextarea(e.target as HTMLTextAreaElement)"
                    ></textarea>
                    
                    <!-- Decorners -->
                    <div class="absolute top-0 left-0 w-1.5 h-1.5 border-t border-l border-primary/30"></div>
                    <div class="absolute bottom-0 right-0 w-1.5 h-1.5 border-b border-r border-primary/30"></div>
                    
                    <div class="absolute bottom-2 right-2 opacity-0 group-focus-within:opacity-100 transition-opacity">
                        <span class="text-[10px] font-mono text-primary animate-pulse">自动保存中...</span>
                    </div>
                </div>

                <div class="mt-8 border-t border-base-content/10 pt-4 pb-8">
                    <div class="flex items-center justify-between mb-4">
                        <div class="text-[10px] text-primary uppercase font-mono tracking-widest">评论</div>
                        <button @click="onFocusComment" class="btn btn-xs btn-outline btn-primary">添加评论</button>
                    </div>
                    <div v-if="comments.length === 0 && isReadyForHeavyContent" class="text-center py-8 opacity-30 text-xs italic">暂无评论</div>
                    
                    <div v-else-if="!isReadyForHeavyContent" class="flex flex-col gap-4">
                         <!-- Comment Skeleton -->
                         <div v-for="i in 2" :key="i" class="flex gap-3">
                             <div class="w-8 h-8 rounded-full bg-base-content/10 animate-pulse"></div>
                             <div class="flex-1 space-y-2">
                                 <div class="h-3 w-1/4 bg-base-content/10 rounded animate-pulse"></div>
                                 <div class="h-10 w-full bg-base-content/10 rounded animate-pulse"></div>
                             </div>
                         </div>
                    </div>

                    <div v-else class="flex flex-col gap-10">
                        <div v-for="thread in organizedComments" :key="thread.id" class="flex flex-col gap-4">
                            <CommentItem :comment="thread" @reply="setReply" />
                            <div v-if="thread.allReplies.length > 0" class="ml-2 flex flex-col gap-3">
                                <CommentItem v-for="reply in thread.allReplies" :key="reply.id" :comment="reply" :is-reply="true" :depth="reply._depth" :parent-author="reply._replyToAuthor" :parent-content="reply._parentContent" @reply="setReply" />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </main>

        <!-- Fixed Bottom Input Footer -->
        <footer class="shrink-0 bg-base-100/95 backdrop-blur border-t border-base-content/10 z-40 pb-[env(safe-area-inset-bottom)] transition-all duration-300">
            <!-- Reply Hint -->
            <div v-if="replyToCommentID" class="px-4 py-2 bg-base-content/5 flex justify-between items-center text-xs font-mono border-b border-base-content/5">
                <span class="text-primary">回复给 @{{ replyToUserName }}</span>
                <button @click="cancelReply" class="w-5 h-5 flex items-center justify-center rounded-full hover:bg-base-content/10"><i class="ri-close-line"></i></button>
            </div>

            <div class="p-3 flex items-end gap-3">
                 <div class="flex-1 bg-base-200/50 rounded-lg border border-base-content/5 focus-within:border-primary/50 focus-within:bg-base-200 transition-all flex items-center">
                    <textarea 
                        v-model="newComment"
                        ref="footerTextarea"
                        rows="1"
                        class="w-full bg-transparent border-none outline-none px-3 py-3 text-sm resize-none max-h-32 placeholder-base-content/30 leading-normal font-sans"
                        :placeholder="replyToCommentID ? `回复 @${replyToUserName}...` : '写下你的评论...'"
                        @input="(e) => resizeTextarea(e.target as HTMLTextAreaElement)"
                        @focus="onFocusComment"
                    ></textarea>
                </div>
                <button 
                    @click="submitComment" 
                    class="w-11 h-11 flex items-center justify-center rounded-full bg-primary text-black shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all text-xl shrink-0"
                    :disabled="!newComment.trim()"
                    :class="{ 'opacity-50 grayscale': !newComment.trim() }"
                >
                    <i class="ri-send-plane-fill"></i>
                </button>
            </div>
        </footer>
    </div>

    <Teleport to="body">
      <div v-if="showDatePicker" class="modal modal-open modal-bottom sm:modal-middle" style="z-index: 9999;">
          <div class="modal-box bg-base-100 border border-primary/20 shadow-[0_0_50px_rgba(0,0,0,0.5)]">
              <div class="flex justify-between items-center mb-6">
                  <h3 class="font-bold text-lg text-primary uppercase">设置截止日期</h3>
                  <button @click="showDatePicker = false" class="btn btn-sm btn-square btn-ghost"><i class="ri-close-line"></i></button>
              </div>
              <div class="flex flex-col gap-6">
                  <div class="form-control w-full">
                      <label class="label"><span class="label-text font-mono text-[10px] uppercase">选择日期与时间</span></label>
                      <input type="datetime-local" v-model="dueAtLocal" class="input input-bordered w-full font-mono bg-base-200/50 rounded-none border-base-content/10">
                  </div>
                  <div class="flex flex-col gap-2">
                      <button @click="updateDueDate" class="btn btn-primary w-full rounded-none">确认设置</button>
                      <button v-if="task?.DueAt" @click="clearDueDate" class="btn btn-ghost w-full rounded-none text-xs text-error/60">清除截止日期</button>
                  </div>
              </div>
          </div>
          <div class="modal-backdrop bg-black/60 backdrop-blur-sm" @click="showDatePicker = false" style="cursor: pointer;"><button>close</button></div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.context-content {
    background-color: #0e1621;
    padding: 16px 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.chat-bubble { max-width: 85%; padding: 6px 12px 8px; font-size: 14px; }
.chat-bubble.other { align-self: flex-start; background-color: #182533; color: #fff; border-radius: 12px 12px 12px 4px; }
.chat-bubble.me { align-self: flex-end; background-color: #2b5278; color: #fff; border-radius: 12px 12px 4px 12px; }
.chat-name { color: #40a7e3; font-size: 13px; margin-bottom: 2px; }
.context-snapshot-section { background: #17212b; border: 1px solid rgba(255, 255, 255, 0.1); border-radius: 12px; overflow: hidden; margin-top: 16px; }
.section-toggle { padding: 12px 16px; background: #17212b; color: #fff; display: flex; justify-content: space-between; align-items: center; }
.history-link { color: #40a7e3; font-size: 12px; }
.detail-title-input { width: 100% !important; margin-bottom: 0 !important; }
.modal { z-index: 2000; }

@keyframes pulse {
    0% { transform: scale(0.95); opacity: 0.5; }
    50% { transform: scale(1.05); opacity: 1; }
    100% { transform: scale(0.95); opacity: 0.5; }
}
</style>
