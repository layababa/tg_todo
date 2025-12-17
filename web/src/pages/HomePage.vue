<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listTasks } from '@/api/task'
import { listDatabases } from '@/api/notion'
import { useAuthStore } from '@/store/auth'
import type { TaskDetail } from '@/types/task'
import type { DatabaseSummary } from '@/types/group'

const router = useRouter()
const authStore = useAuthStore()

// State
const tasks = ref<TaskDetail[]>([])
const loading = ref(true)
const error = ref('')
const currentTab = ref<'assigned' | 'created' | 'all'>('assigned')
const filterOpen = ref(false)

// Filter State
const databases = ref<DatabaseSummary[]>([])
const selectedDbId = ref<string>('') // Empty string = All

// Computed
const user = computed(() => authStore.user)
const userName = computed(() => user.value?.name || 'User')
const activeTasks = computed(() => {
    // tasks.value is already filtered by backend via fetchTasks
    return tasks.value.filter(t => t.task.Status !== 'Done')
})
const doneTasks = computed(() => tasks.value.filter(t => t.task.Status === 'Done'))
const showDone = ref(false)
const toggleDone = () => {
  showDone.value = !showDone.value
}

const selectedDbName = computed(() => {
    if (!selectedDbId.value) return 'All'
    const db = databases.value.find(d => d.id === selectedDbId.value)
    return db ? db.name : 'Unknown'
})

// Methods
const fetchTasks = async () => {
  loading.value = true
  try {
    const viewMap: Record<string, string> = {
        'assigned': 'assigned',
        'created': 'created',
        'all': 'all'
    }
    const viewIdx = viewMap[currentTab.value] || 'assigned'
    
    tasks.value = await listTasks({ 
        view: viewIdx,
        database_id: selectedDbId.value || undefined
    })
  } catch (e: any) {
    error.value = 'Data sync failed'
    console.error(e)
  } finally {
    loading.value = false
  }
}

const fetchDatabases = async () => {
    try {
        databases.value = await listDatabases()
    } catch (e) {
        console.error("Failed to load databases", e)
    }
}

watch([currentTab, selectedDbId], () => {
  fetchTasks()
})

const selectDb = (id: string) => {
    selectedDbId.value = id
    filterOpen.value = false
}

const goToDetail = (id: string) => router.push(`/tasks/${id}`)
const goToSettings = () => router.push('/settings')
const formatDate = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

onMounted(() => {
    fetchTasks()
    fetchDatabases()
    if (!authStore.user) authStore.fetchStatus()
})
</script>

<template>
  <div class="grid-bg"></div>
  <div class="scan-line"></div>

  <div class="app-container">
    <!-- Header -->
    <header class="header sticky top-0 z-30">
        <div class="flex justify-between items-center mb-6">
            <div class="font-mono text-[10px] text-primary border border-primary px-1.5 py-0.5 tracking-widest">系统.V2.0</div>
            <div class="flex gap-3">
                <button class="icon-btn tech-btn" :class="{ '!text-primary !border-primary': filterOpen || selectedDbId }" aria-label="Filter" @click="filterOpen = !filterOpen">
                    <i class="ri-filter-3-line"></i>
                </button>
                <button class="icon-btn tech-btn" aria-label="Settings" @click="goToSettings">
                    <i class="ri-settings-4-line"></i>
                </button>
            </div>
        </div>

        <div class="mb-6">
            <h1 class="text-3xl font-light mb-2 tracking-tight">你好, <span class="font-bold">{{ userName }}</span></h1>
            <div class="flex items-center gap-1.5 font-mono text-[10px] text-base-content/60">
                <span class="w-1.5 h-1.5 rounded-full bg-primary shadow-[0_0_6px_var(--primary)] animate-pulse"></span> 系统在线
            </div>
        </div>

        <!-- Tech Tabs -->
        <div class="border-b border-base-content/10">
            <div class="flex relative">
                <button 
                    @click="currentTab = 'assigned'"
                    class="flex-1 bg-transparent border-none text-base-content/60 font-mono text-xs py-3 cursor-pointer transition-colors"
                    :class="{ 'text-primary font-bold': currentTab === 'assigned' }"
                >指派给我</button>
                <button 
                    @click="currentTab = 'created'"
                    class="flex-1 bg-transparent border-none text-base-content/60 font-mono text-xs py-3 cursor-pointer transition-colors"
                    :class="{ 'text-primary font-bold': currentTab === 'created' }"
                >我创建的</button>
                 <button 
                    @click="currentTab = 'all'"
                    class="flex-1 bg-transparent border-none text-base-content/60 font-mono text-xs py-3 cursor-pointer transition-colors"
                    :class="{ 'text-primary font-bold': currentTab === 'all' }"
                >全部任务</button>
                
                <!-- Active Line Indicator -->
                <div class="absolute bottom-[-1px] left-0 w-1/3 h-0.5 bg-primary shadow-[0_-2px_8px_rgba(171,246,0,0.2)] transition-transform duration-300"
                    :style="{ transform: `translateX(${currentTab === 'assigned' ? 0 : currentTab === 'created' ? 100 : 200}%)` }"
                ></div>
            </div>
        </div>
    </header>

    <!-- Filter Bar (Selector) -->
    <div v-show="filterOpen" class="mx-5 mb-4 p-3 border border-dashed border-base-content/20 bg-base-200/90 backdrop-blur rounded-lg z-20 shadow-lg">
        <div class="text-[10px] font-mono mb-2 opacity-50 uppercase tracking-widest">Select Database</div>
        <div class="flex flex-wrap gap-2">
            <div 
                @click="selectDb('')"
                class="px-3 py-1.5 text-xs border rounded cursor-pointer transition-colors"
                :class="!selectedDbId ? 'bg-primary text-black border-primary font-bold' : 'border-base-content/10 hover:border-primary/50'"
            >
                All
            </div>
            <div 
                v-for="db in databases" 
                :key="db.id"
                @click="selectDb(db.id)"
                class="px-3 py-1.5 text-xs border rounded cursor-pointer transition-colors flex items-center gap-1"
                :class="selectedDbId === db.id ? 'bg-primary text-black border-primary font-bold' : 'border-base-content/10 hover:border-primary/50'"
            >
                <i v-if="db.icon" :class="db.icon"></i>
                {{ db.name }}
            </div>
        </div>
    </div>

    <!-- Active Filter Indicator (When closed but active) -->
    <div v-if="!filterOpen && selectedDbId" class="mx-5 mb-4 flex justify-between items-center font-mono text-[11px]">
        <div class="flex items-center gap-2 text-primary">
            <i class="ri-filter-fill"></i>
            <span>Filter: <span class="font-bold underline">{{ selectedDbName }}</span></span>
        </div>
        <button class="text-base-content/40 hover:text-base-content" @click="selectDb('')">Clear</button>
    </div>

    <!-- Task List -->
    <main class="px-0 pb-24" id="taskList">
        <div v-if="loading && tasks.length === 0" class="flex flex-col gap-4 p-4">
             <div v-for="i in 3" :key="i" class="skeleton h-24 w-full rounded bg-base-200/50"></div>
        </div>

        <div v-else-if="tasks.length === 0" class="text-center py-10 opacity-50">
            <i class="ri-inbox-line text-5xl mb-4 block"></i>
            <p>暂无任务</p>
        </div>

        <template v-else>
            <!-- Active Tasks -->
            <div v-if="activeTasks.length > 0">
                <div class="px-5 py-3 text-[10px] font-mono text-primary uppercase tracking-widest flex items-center justify-between">
                    待办事项 <span class="bg-base-content/10 px-1.5 py-0.5 rounded">{{ activeTasks.length }}</span>
                </div>
                
                <div 
                    v-for="{ task } in activeTasks" 
                    :key="task.ID" 
                    @click="goToDetail(task.ID)"
                    class="mx-5 mb-3 bg-base-200/60 border border-base-content/10 pl-3 relative overflow-hidden transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[0_4px_20px_rgba(171,246,0,0.1)] hover:border-primary group cursor-pointer"
                    style="clip-path: polygon(0 0, 100% 0, 100% calc(100% - 16px), calc(100% - 16px) 100%, 0 100%); border-left-width: 4px;"
                >
                    <div class="p-4 pr-3">
                        <div class="flex justify-between items-start mb-3">
                            <div class="text-sm font-medium leading-snug pr-2">{{ task.Title }}</div>
                            <div class="font-mono text-[10px] opacity-60 border border-base-content/20 px-1 rounded truncate max-w-[80px]">
                                {{ task.DatabaseID ? databases.find(d => d.id === task.DatabaseID)?.name || 'DB' : 'Manual' }}
                            </div>
                        </div>
                        <div class="flex items-center gap-4 text-xs text-base-content/60 font-mono">
                            <div class="flex items-center gap-1.5">
                                <i class="ri-user-3-line"></i>
                                <span>{{ task.Assignees?.[0]?.FirstName || 'Me' }}</span>
                            </div>
                            <div class="flex items-center gap-1.5">
                                <i class="ri-time-line"></i>
                                <span>{{ formatDate(task.CreatedAt) }}</span>
                            </div>
                        </div>
                    </div>
                    <!-- Hover Corner Effect -->
                    <div class="absolute bottom-0 right-0 w-4 h-4 bg-gradient-to-tl from-base-content/30 to-transparent opacity-50 z-10 pointer-events-none group-hover:from-primary group-hover:opacity-100 transition-all"></div>
                </div>
            </div>

            <!-- Done Tasks -->
             <div v-if="doneTasks.length > 0" class="mt-6">
                 <div @click="toggleDone" class="px-5 py-3 flex items-center gap-2 cursor-pointer select-none opacity-60 hover:opacity-100 transition-opacity">
                     <span class="text-xs">已完成 ({{ doneTasks.length }})</span>
                     <i class="ri-arrow-down-s-line transition-transform duration-300" :class="{ 'rotate-180': showDone }"></i>
                 </div>

                 <div v-show="showDone">
                    <div 
                        v-for="{ task } in doneTasks" 
                        :key="task.ID" 
                        @click="goToDetail(task.ID)"
                        class="mx-5 mb-3 bg-base-200/30 border border-base-content/5 pl-3 opacity-60 grayscale transition-all hover:grayscale-0 hover:opacity-100 cursor-pointer"
                        style="clip-path: polygon(0 0, 100% 0, 100% calc(100% - 16px), calc(100% - 16px) 100%, 0 100%); border-left: 4px solid var(--border-dim);"
                    >
                        <div class="p-4">
                             <div class="text-sm line-through opacity-70">{{ task.Title }}</div>
                        </div>
                    </div>
                 </div>
             </div>
        </template>
    </main>

    <!-- FAB -->
    <button class="fixed bottom-6 right-6 w-14 h-14 bg-primary text-black rounded-none flex items-center justify-center text-2xl shadow-[0_0_20px_rgba(171,246,0,0.4)] transition-transform hover:scale-105 active:scale-95 z-40"
        style="clip-path: polygon(10px 0, 100% 0, 100% calc(100% - 10px), calc(100% - 10px) 100%, 0 100%, 0 10px);"
        @click="goToDetail('new')"
    >
        <i class="ri-add-line"></i>
    </button>

  </div>
</template>
