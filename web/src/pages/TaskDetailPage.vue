<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getTask, patchTask, createTask, deleteTask } from '@/api/task'
import { listComments, createComment, type Comment } from '@/api/comment'
import CommentItem from '@/components/CommentItem.vue'
import { useAuthStore } from '@/store/auth'
import type { Task } from '@/types/task'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// State
const task = ref<Task | null>(null)
const comments = ref<Comment[]>([])
const loading = ref(true)
const errorMessage = ref('')
const contextOpen = ref(false)
const hudCollapsed = ref(true)
const newComment = ref('')
const isNew = computed(() => route.params.id === 'new')
const isSaving = ref(false)
const replyToCommentID = ref<string | null>(null)
const replyToUserName = ref<string>('')

// Computed
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

    // 1. Initialize map and sanitize data
    comments.value.forEach(c => {
        map.set(c.id, { ...c, children: [] })
    })

    // 2. Build tree and identify root comments
    // Sort all by date first to ensure stable processing
    const sortedRaw = [...comments.value].sort((a, b) => 
        new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    )

    sortedRaw.forEach(c => {
        const comment = map.get(c.id)
        if (c.parent_id && map.has(c.parent_id)) {
            map.get(c.parent_id).children.push(comment)
        } else {
            // If no parent_id OR parent not found in current set, treat as root
            rootComments.push(comment)
        }
    })

    // 3. Helper for DFS flattening
    const flatten = (comment: any, targetList: any[], depth: number) => {
        // Sort children by date before recursion to keep sub-thread chronological
        comment.children.sort((a: any, b: any) => 
            new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
        )

        comment.children.forEach((child: any) => {
            child._replyToAuthor = comment.user?.name || 'Unknown'
            child._parentContent = comment.content // 抓取父评论内容用于引用
            child._depth = depth // Track depth for styling
            targetList.push(child)
            flatten(child, targetList, depth + 1) // Recurse
        })
    }

    // 4. Final threads construction
    const finalThreads = rootComments.map(root => {
        const allReplies: any[] = []
        flatten(root, allReplies, 1) // Replies start at depth 1
        return {
            ...root,
            allReplies
        }
    })

    // Sort threads: Newest root comment at the top
    return finalThreads.sort((a, b) => 
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    )
})

// Methods
const fetchData = async () => {
    if (isNew.value) {
        // Init empty task for creation
        task.value = {
            ID: '',
            Title: '',
            Status: 'To Do',
            SyncStatus: 'Pending',
            CreatedAt: new Date().toISOString(),
            UpdatedAt: new Date().toISOString(),
            Snapshots: [],
            Assignees: []
        } as Task
        loading.value = false
        // Auto focus title
        nextTick(() => {
             document.querySelector('textarea')?.focus()
        })
        return
    }

    try {
        const id = route.params.id as string
        const [t, c] = await Promise.all([
            getTask(id),
            listComments(id)
        ])
        task.value = t
        comments.value = c || []
        loading.value = false
    } catch (e) {
        errorMessage.value = 'Failed to load task data'
        loading.value = false
    }
}

const saveTask = async () => {
    if (!task.value || !task.value.Title.trim() || isSaving.value) return
    isSaving.value = true
    try {
        if (isNew.value) {
            const created = await createTask({ 
                title: task.value.Title,
                description: task.value.Description 
            })
            // Redirect to actual ID
            router.replace(`/tasks/${created.ID}`)
            // Update local state
            task.value = created
        } else {
            // Update existing
             await patchTask(task.value.ID, { 
                title: task.value.Title,
                // description: task.value.Description // Backend API supports this? Yes, now it does.
             })
        }
    } catch (e) {
        console.error('Failed to save task', e)
        alert('Failed to save task')
    } finally {
        isSaving.value = false
    }
}

const updateTitle = async () => {
    if (isNew.value) return // Don't auto-save on blur for new task, wait for button or Enter? 
    if (task.value?.Title) {
        saveTask()
    }
}

const cycleStatus = async () => {
    if (!task.value || !task.value.ID) return
    
    const statusMap: Record<string, string> = {
        'To Do': 'In Progress',
        'In Progress': 'Done',
        'Done': 'To Do'
    }
    const currentStatus = task.value.Status || 'To Do'
    const nextStatus = statusMap[currentStatus] || 'To Do'
    
    const originalStatus = task.value.Status
    task.value.Status = nextStatus as any
    
    try {
        await patchTask(task.value.ID, { status: nextStatus as any })
    } catch (e) {
        console.error('Failed to update status', e)
        task.value.Status = originalStatus
    }
}

const submitComment = async () => {
    if (!newComment.value.trim() || !task.value?.ID) return
    const content = newComment.value
    const parentID = replyToCommentID.value
    newComment.value = '' 
    replyToCommentID.value = null
    replyToUserName.value = ''
    try {
        const comment = await createComment(task.value.ID, { 
            content,
            parent_id: parentID || undefined
        })
        
        // Ensure user data is present (already handled by backend preload, but safety first)
        if (!comment.user && authStore.user) {
            comment.user = {
                id: authStore.user.id,
                name: authStore.user.name,
                photo_url: authStore.user.photo_url
            }
        }
        
        comments.value.push(comment)
        hudCollapsed.value = true
    } catch (e) {
        console.error('Failed to create comment', e)
        newComment.value = content 
        replyToCommentID.value = parentID
    }
}

const setReply = (comment: Comment) => {
    replyToCommentID.value = comment.id
    replyToUserName.value = comment.user?.name || 'Unknown'
    hudCollapsed.value = false
    nextTick(() => {
        document.querySelector('.hud-expanded-content textarea')?.focus()
    })
}

const cancelReply = () => {
    replyToCommentID.value = null
    replyToUserName.value = ''
}

const onDelete = async () => {
    if (!confirm('Are you sure you want to delete this task?')) return
    try {
        if (task.value?.ID) {
            await deleteTask(task.value.ID)
        }
        router.push('/home')
    } catch (e) {
        alert('Failed to delete task')
    }
}

const toggleContext = () => contextOpen.value = !contextOpen.value
const toggleHud = () => hudCollapsed.value = !hudCollapsed.value
const onFocusComment = () => hudCollapsed.value = false
const goBack = () => router.push('/home')

// Simple Description ContentEditable Binding
const onDescriptionBlur = (e: Event) => {
    const target = e.target as HTMLElement
    if (task.value) {
        task.value.Description = target.innerText
    }
}

onMounted(fetchData)
</script>

<template>
  <!-- Backgrounds from main.css -->
  <div class="grid-bg"></div>
  <div class="scan-line"></div>

  <div class="app-container relative">
      <!-- Header -->
      <header class="header sticky top-0 z-30">
          <div class="flex items-center justify-between">
              <button @click="goBack" class="icon-btn tech-btn !w-auto !px-2 !border-none text-sm gap-2">
                  <i class="ri-arrow-left-line"></i> 返回
              </button>
              
              <!-- Notion & More Actions -->
              <div class="flex items-center gap-2">
                  <!-- Open in Notion -->
                  <a v-if="task?.NotionURL" :href="task.NotionURL" target="_blank" class="icon-btn tech-btn !w-auto !px-2 !border-none text-sm gap-2">
                      <i class="ri-notion-fill text-white"></i> Notion
                  </a>

                  <!-- More Actions (Delete) -->
                  <div v-if="!isNew" class="dropdown dropdown-end">
                    <button tabindex="0" class="icon-btn tech-btn !border-none"><i class="ri-more-2-fill"></i></button>
                    <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52 border border-base-content/10">
                        <li><a @click="onDelete" class="text-error"><i class="ri-delete-bin-line"></i> Delete Task</a></li>
                    </ul>
                  </div>
                  <div v-else>
                     <button @click="saveTask" class="btn btn-sm btn-primary" :disabled="isSaving || !task?.Title">
                        {{ isSaving ? 'Saving...' : 'Create' }}
                     </button>
                  </div>
              </div>
          </div>
      </header>

      <!-- Content -->
      <main class="px-4 pb-48 pt-4">
          <div v-if="loading" class="flex justify-center mt-20">
              <span class="loading loading-spinner text-primary"></span>
          </div>

          <div v-else-if="task">
              <!-- Title Area -->
              <div class="flex items-start justify-between gap-4 mb-6">
                  <textarea 
                    v-model="task.Title" 
                    @blur="updateTitle"
                    placeholder="Task Title..."
                    class="detail-title-input flex-1 bg-transparent border-b border-transparent focus:border-primary text-2xl font-bold font-display outline-none resize-none overflow-hidden placeholder-base-content/30" 
                    rows="1"
                  ></textarea>
                  
                  <!-- TG Jump Button - Redesigned Tech Style -->
                  <a v-if="task?.ChatJumpURL" :href="task.ChatJumpURL" target="_blank" 
                     class="flex items-center gap-2 px-3 py-2 bg-[#24A1DE]/5 border border-[#24A1DE]/20 hover:border-[#24A1DE] hover:bg-[#24A1DE]/10 transition-all shrink-0 mt-1 group"
                     title="跳转到 Telegram 会话"
                  >
                      <i class="ri-telegram-fill text-[#24A1DE] text-lg group-hover:scale-110 transition-transform"></i>
                      <span class="text-[10px] font-mono font-bold text-[#24A1DE] tracking-widest">消息直达</span>
                  </a>
              </div>

              <!-- Context Snapshot -->
              <div v-if="contextItems.length > 0" class="context-snapshot-section">
                  <div @click="toggleContext" class="section-toggle cursor-pointer select-none">
                      <div class="flex items-center gap-2 text-sm font-mono">
                          <i class="ri-chat-history-line"></i>
                          <span>上下文快照 ({{ contextItems.length }}条)</span>
                      </div>
                      <div class="flex items-center gap-2">
                          <a v-if="task?.ChatJumpURL" :href="task.ChatJumpURL" target="_blank" class="history-link hover:underline">查看完整记录 <i class="ri-external-link-line"></i></a>
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

              <!-- Editor / Comments -->
              <div class="editor-content tech-border p-4 rounded-lg mt-6 min-h-[100px] outline-none">
                  <p class="text-white/50 italic text-sm mb-4" v-if="!task.Description">暂无描述...</p>
                  <p v-else class="text-sm">{{ task.Description }}</p>

                  <!-- Comments Section -->
                  <div class="mt-8 border-t border-base-content/10 pt-4">
                      <div class="flex items-center justify-between mb-4">
                          <div class="text-[10px] text-primary uppercase font-mono tracking-widest">评论</div>
                          <button @click="onFocusComment" class="btn btn-xs btn-outline btn-primary font-mono text-[10px]">添加评论</button>
                      </div>

                      <div v-if="comments.length === 0" class="text-center py-8 opacity-30 text-xs italic">
                          暂无评论
                      </div>

                      <div v-else class="flex flex-col gap-10">
                          <!-- Discussion Threads -->
                          <div v-for="thread in organizedComments" :key="thread.id" class="flex flex-col gap-4">
                              <!-- Root Comment -->
                              <CommentItem 
                                :comment="thread" 
                                @reply="setReply"
                              />
                              
                              <!-- Flattened Replies Area - Minimal offset, no rail -->
                              <div v-if="thread.allReplies.length > 0" class="ml-2 flex flex-col gap-3">
                                  <CommentItem 
                                    v-for="reply in thread.allReplies" 
                                    :key="reply.id" 
                                    :comment="reply" 
                                    :is-reply="true"
                                    :depth="reply._depth"
                                    :parent-author="reply._replyToAuthor"
                                    :parent-content="reply._parentContent"
                                    @reply="setReply"
                                  />
                              </div>
                          </div>
                      </div>
                  </div>
              </div>
          </div>
      </main>
  </div>

  <!-- HUD Overlay - Darkens background when HUD is active -->
  <div 
    v-show="!hudCollapsed" 
    @click="hudCollapsed = true"
    class="hud-overlay fixed inset-0 bg-black/60 backdrop-blur-[2px] z-[900] transition-opacity duration-300"
  ></div>

  <!-- HUD Cockpit - Moved outside app-container for true fixed positioning -->
  <div class="hud-container-fixed">
    <div class="bottom-hud" :class="{ 'hud-collapsed': hudCollapsed }">
        <!-- Toggle -->
        <div @click="toggleHud" class="hud-toggle-btn cursor-pointer flex justify-center py-1">
            <i class="ri-arrow-down-s-line transition-transform duration-300" :class="{ 'rotate-180': hudCollapsed }"></i>
        </div>

        <div v-show="!hudCollapsed" class="hud-expanded-content p-4 border-b border-base-content/5">
            <!-- Row 1: Properties (Status, Assignee, Date) -->
            <div class="flex flex-wrap items-center gap-2 mb-3">
                <div @click="cycleStatus" class="badge badge-outline gap-1.5 h-8 px-3 cursor-pointer hover:bg-white hover:text-black transition-all select-none border-base-content/20 rounded-none">
                    <i class="ri-loader-2-line text-xs"></i> 
                    <span class="text-xs font-bold">{{ task?.Status || '待办' }}</span>
                </div>
                <div class="badge badge-outline gap-1.5 h-8 px-3 border-base-content/20 rounded-none">
                    <i class="ri-user-3-line text-xs"></i> 
                    <span class="text-xs">我</span>
                </div>
                <div class="badge badge-outline gap-1.5 h-8 px-3 border-base-content/20 rounded-none text-xs">
                    <i class="ri-calendar-event-line"></i> 截止日期
                </div>
            </div>

            <!-- Row 2: Link Actions (Now start-aligned and consistent style) -->
            <div class="flex gap-2 mb-4">
                <a v-if="task?.ChatJumpURL" :href="task.ChatJumpURL" target="_blank" class="badge badge-outline gap-1.5 h-8 px-3 cursor-pointer hover:bg-[#24A1DE]/10 hover:border-[#24A1DE]/50 transition-all text-[#24A1DE] border-base-content/20 rounded-none">
                    <i class="ri-telegram-fill"></i>
                    <span class="text-[10px] font-mono uppercase">Chat</span>
                </a>
                <a :href="`https://t.me/${authStore.user?.telegram_username || ''}`" target="_blank" class="badge badge-outline gap-1.5 h-8 px-3 cursor-pointer hover:bg-primary/10 hover:border-primary/50 transition-all text-primary border-base-content/20 rounded-none">
                    <i class="ri-user-voice-line"></i>
                    <span class="text-[10px] font-mono uppercase">PM</span>
                </a>
                <a 
                    :href="task?.NotionURL || '#'" 
                    target="_blank" 
                    class="badge badge-outline gap-1.5 h-8 px-3 transition-all rounded-none"
                    :class="[task?.NotionURL ? 'cursor-pointer hover:bg-white/10 hover:border-white/50 text-white border-base-content/20' : 'opacity-10 grayscale pointer-events-none border-transparent']"
                >
                    <i class="ri-notion-fill"></i>
                    <span class="text-[10px] font-mono uppercase">Notion</span>
                </a>
            </div>
            
            <!-- Input Area -->
            <div class="flex flex-col gap-3">
                <!-- Reply Indicator -->
                <div v-if="replyToCommentID" class="flex items-center justify-between bg-primary/10 border-l-2 border-primary px-3 py-1.5 rounded-none">
                    <div class="text-[10px] text-primary font-mono truncate">
                        REPLYING TO <span class="font-bold">@{{ replyToUserName }}</span>
                    </div>
                    <button @click="cancelReply" class="text-primary hover:text-white transition-colors">
                        <i class="ri-close-line"></i>
                    </button>
                </div>

                <textarea 
                    v-model="newComment"
                    @focus="onFocusComment"
                    :placeholder="replyToCommentID ? `回复 ${replyToUserName}...` : '输入评论内容...'" 
                    class="w-full bg-base-300/30 rounded-none p-3 text-sm focus:outline-none focus:ring-1 focus:ring-primary/30 border border-base-content/10 h-32 resize-none transition-all duration-300"
                ></textarea>
                
                <button 
                    @click="submitComment"
                    class="btn btn-primary rounded-none w-full h-10 min-h-0 font-bold tracking-widest text-xs"
                    :disabled="!newComment.trim()"
                >
                    <i class="ri-send-plane-fill mr-2"></i>
                    发送评论
                </button>
            </div>
        </div>

        <!-- Collapsed State: Global Command Bar -->
        <div v-show="hudCollapsed" class="bg-base-200/30 p-5 flex items-center gap-4 group cursor-text" @click="onFocusComment">
             <div class="w-2.5 h-2.5 rounded-full bg-primary animate-pulse shadow-[0_0_8px_var(--primary)]"></div>
             <input 
                readonly
                type="text" 
                placeholder="输入 / 执行系统指令..." 
                class="bg-transparent w-full text-sm font-mono outline-none text-primary/70 placeholder-primary/30 pointer-events-none"
             >
             <i class="ri-command-line text-primary/20 group-hover:text-primary/50 transition-colors"></i>
        </div>
    </div>
  </div>
</template>

<style scoped>
/* HUD Transition and Layout Optimization */
.bottom-hud {
    transition: all 0.5s cubic-bezier(0.16, 1, 0.3, 1);
    background: rgba(10, 10, 10, 0.98);
    backdrop-filter: blur(25px);
    -webkit-backdrop-filter: blur(25px);
    border-top: 1px solid rgba(171, 246, 0, 0.2);
    box-shadow: 0 -20px 60px rgba(0, 0, 0, 1);
    width: 100%;
    max-width: 600px; /* Match container */
    margin: 0 auto;
    overflow: hidden;
    padding-bottom: max(12px, env(safe-area-inset-bottom));
}

.hud-expanded-content {
    animation: HUDReveal 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    transform-origin: bottom;
}

@keyframes HUDReveal {
    from {
        opacity: 0;
        transform: translateY(40px) scaleY(0.95);
        max-height: 0;
    }
    to {
        opacity: 1;
        transform: translateY(0) scaleY(1);
        max-height: 800px; /* Enough for content */
    }
}

.hud-toggle-btn {
    z-index: 110;
    border-color: rgba(171, 246, 0, 0.25);
    background: #0a0a0a;
    box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.5);
    border-radius: 0; /* Match sharp theme */
}

.hud-container-fixed {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 1000;
    display: flex;
    justify-content: center;
    pointer-events: none; /* Let events pass to underlying container */
}

.bottom-hud {
    pointer-events: auto; /* Re-enable events for HUD itself */
}

/* Telegram Style Chat Log for Context Snapshot (Redone for consistency) */
.context-content {
    background-color: #0e1621;
    padding: 16px 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.chat-bubble {
    max-width: 85%;
    padding: 6px 12px 8px;
    position: relative;
    font-size: 14px;
    line-height: 1.4;
    word-wrap: break-word;
    margin-bottom: 2px;
}

.chat-bubble.other {
    align-self: flex-start;
    background-color: #182533;
    color: #fff;
    border-radius: 12px 12px 12px 4px;
}

.chat-bubble.me {
    align-self: flex-end;
    background-color: #2b5278;
    color: #fff;
    border-radius: 12px 12px 4px 12px;
}

.chat-name {
    font-weight: 700;
    font-size: 13px;
    color: #40a7e3;
    margin-bottom: 2px;
}

/* Ensure context snapshot section looks solid */
.context-snapshot-section {
    background: #17212b;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    overflow: hidden;
    margin-top: 16px;
}

.section-toggle {
    padding: 12px 16px;
    background: #17212b;
    color: #fff;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.history-link {
    color: #40a7e3;
    font-size: 12px;
    margin-right: 8px;
    text-decoration: none;
}

.detail-title-input {
    width: 100% !important;
    margin-bottom: 0 !important;
}
</style>
