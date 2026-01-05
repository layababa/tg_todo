<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { useSwipeBack } from '@/composables/useSwipeBack'
import { useSafeArea } from '@/composables/useSafeArea'
import { listGroups } from '@/api/group'
import { listDatabases } from '@/api/notion'
import { getMe, updateSettings, generateCalendarToken } from '@/api/user'
import type { Group, DatabaseSummary } from '@/types/group'
import type { UserProfile } from '@/types/auth'

const router = useRouter()
const authStore = useAuthStore()

// Safe Area
const { safeAreaTop } = useSafeArea()

// Add swipe back support
useSwipeBack()

// State
const userProfile = ref<UserProfile | null>(null)
const groups = ref<Group[]>([])
const databases = ref<DatabaseSummary[]>([])
const loading = ref(true)

// Computed
const user = computed(() => userProfile.value || authStore.user)
const userName = computed(() => user.value?.name || 'User')
const userInitial = computed(() => userName.value.charAt(0).toUpperCase())
const notionConnected = computed(() => user.value?.notion_connected ?? authStore.notionConnected)

const defaultDbName = computed(() => {
    const dbId = user.value?.default_database_id
    if (!dbId) return '未设置'
    const db = databases.value.find(d => d.id === dbId)
    return db?.name || 'Unknown DB'
})

const timezone = computed(() => user.value?.timezone || 'UTC+0')

// Calendar Sync
const calendarUrl = ref('')
const calendarLoading = ref(false)

const enableCalendarSync = async () => {
    calendarLoading.value = true
    try {
        const result = await generateCalendarToken()
        if (result?.webcal_url) {
            calendarUrl.value = result.webcal_url
            // Try to open webcal URL directly (will prompt calendar app)
            window.location.href = result.webcal_url
        }
    } catch (e) {
        console.error('Failed to generate calendar token', e)
    } finally {
        calendarLoading.value = false
    }
}

// Methods
const initData = async () => {
    loading.value = true
    try {
        const [profile, groupList, dbList] = await Promise.all([
            getMe().catch(() => null), // Fail safe
            listGroups().catch(() => []),
            listDatabases().catch(() => [])
        ])
        if (profile) userProfile.value = profile
        groups.value = groupList || []
        databases.value = dbList || []
    } catch (e) {
        console.error('Failed to init settings', e)
    } finally {
        loading.value = false
    }
}

const goBack = () => router.push('/home')
const logout = () => {
    // Basic mock logout
    localStorage.clear()
    window.location.reload()
}

const showDbPicker = ref(false)

const onDbClick = () => {
    showDbPicker.value = true
}

const selectDb = async (id: string) => {
    try {
        const updated = await updateSettings({ default_database_id: id })
        if (updated) {
            userProfile.value = updated
            // Also update store to reflect globally if needed, though we rely on profile ref here
            // authStore.setUser(updated) 
        }
        showDbPicker.value = false
    } catch (e) {
        console.error('Failed to set default db', e)
    }
}

const onTimezoneClick = () => {
    // Mock Toggle for testing PATCH
    if (!userProfile.value) return
    const newTz = userProfile.value.timezone === 'UTC+8' ? 'UTC+0' : 'UTC+8'
    updateSettings({ timezone: newTz }).then(updated => {
        userProfile.value = updated
    })
}

onMounted(() => {
    if (!authStore.user) authStore.fetchStatus()
    initData()
})
</script>

<template>
  <div class="page-root">
    <div class="grid-bg"></div>
    <div class="scan-line"></div>

    <div class="app-container">
        <!-- Header -->
        <header class="header" :style="{ paddingTop: safeAreaTop + 'px' }">
            <div class="flex justify-between items-center mb-10">
                <a @click.prevent="goBack" href="#" class="flex items-center gap-2 text-white no-underline font-mono text-xs cursor-pointer hover:text-primary transition-colors">
                    <i class="ri-arrow-left-line"></i> 返回首页
                </a>
                <div class="flex gap-3">
                    <button @click="logout" class="icon-btn tech-btn" aria-label="注销">
                        <i class="ri-logout-box-r-line"></i>
                    </button>
                </div>
            </div>

            <div class="text-center mb-10">
                <div class="w-20 h-20 mx-auto mb-5 rounded-full border border-primary flex items-center justify-center bg-base-200 shadow-[0_0_30px_rgba(171,246,0,0.1)] relative">
                    <span class="text-3xl font-bold text-primary">{{ userInitial }}</span>
                    <!-- Spinner if avatar loading could go here -->
                    <div class="absolute inset-[-5px] border border-dashed border-base-content/20 rounded-full animate-[spin_10s_linear_infinite]"></div>
                </div>
                <h2 class="text-2xl font-light mb-2 font-display">{{ userName }}</h2>
                <div class="flex items-center justify-center gap-2 text-xs font-mono text-base-content/60">
                    <span class="w-1.5 h-1.5 rounded-full shadow-[0_0_6px]" :class="notionConnected ? 'bg-primary shadow-primary' : 'bg-error shadow-error'"></span> 
                    {{ notionConnected ? '已连接 Notion' : '未连接 Notion' }}
                </div>
            </div>
        </header>

        <div class="px-5 pb-24">
            <!-- Personal Config -->
            <div class="mb-8">
                <div class="text-[10px] text-primary mb-4 uppercase font-mono tracking-widest">个人配置</div>

                <div @click="onDbClick" class="flex justify-between items-center p-4 bg-base-200/50 border border-base-content/10 mb-2 cursor-pointer hover:border-base-content/30 transition-colors">
                    <div class="flex items-center gap-2 text-base-content/60 font-mono text-xs">
                        <i class="ri-inbox-archive-line"></i> 默认收集箱
                    </div>
                    <div class="flex items-center gap-2">
                        <span class="text-xs px-2 py-0.5 border border-primary text-primary bg-primary/5 rounded-[2px] font-mono">{{ defaultDbName }}</span>
                        <i class="ri-arrow-right-s-line text-base-content/40"></i>
                    </div>
                </div>

                <div @click="onTimezoneClick" class="flex justify-between items-center p-4 bg-base-200/50 border border-base-content/10 mb-2 cursor-pointer hover:border-base-content/30 transition-colors">
                    <div class="flex items-center gap-2 text-base-content/60 font-mono text-xs">
                        <i class="ri-global-line"></i> 时区设置
                    </div>
                    <div class="flex items-center gap-2">
                        <span class="font-mono text-xs">{{ timezone }}</span>
                        <i class="ri-arrow-right-s-line text-base-content/40"></i>
                    </div>
                </div>
            </div>

            <!-- Connection Management -->
            <div class="mb-8">
                <div class="text-[10px] text-primary mb-4 uppercase font-mono tracking-widest">连接管理</div>

                <!-- Groups Link -->
                <div @click="router.push('/groups')" class="flex justify-between items-center p-4 bg-base-200/50 border border-base-content/10 mb-2 cursor-pointer hover:border-base-content/30 transition-colors">
                    <div class="flex items-center gap-2 text-base-content/60 font-mono text-xs">
                        <i class="ri-group-line"></i> 管理群组绑定
                    </div>
                    <div class="flex items-center gap-2">
                        <span v-if="!loading" class="bg-base-content/20 text-xs px-2 py-0.5 rounded-full min-w-[20px] text-center">{{ groups.length }}</span>
                        <span v-else class="loading loading-spinner loading-xs"></span>
                        <i class="ri-arrow-right-s-line text-base-content/40"></i>
                    </div>
                </div>

                <div class="flex justify-between items-center p-4 bg-base-200/50 border border-base-content/10 mb-2 cursor-pointer hover:border-base-content/30 transition-colors">
                    <div class="flex items-center gap-2 text-base-content/60 font-mono text-xs">
                        <i class="ri-refresh-line"></i> 刷新字段缓存
                    </div>
                    <div class="flex items-center gap-2">
                        <i class="ri-arrow-right-s-line text-base-content/40"></i>
                    </div>
                </div>
            </div>

            <!-- Calendar Sync -->
            <div class="mb-8">
                <div class="text-[10px] text-primary mb-4 uppercase font-mono tracking-widest">日历同步</div>

                <div v-if="!calendarUrl" @click="enableCalendarSync" class="flex justify-between items-center p-4 bg-base-200/50 border border-base-content/10 mb-2 cursor-pointer hover:border-primary transition-colors">
                    <div class="flex items-center gap-2 text-base-content/60 font-mono text-xs">
                        <i class="ri-calendar-line"></i> 添加日历订阅
                    </div>
                    <div class="flex items-center gap-2">
                        <span v-if="calendarLoading" class="loading loading-spinner loading-xs"></span>
                        <i v-else class="ri-add-line text-primary"></i>
                    </div>
                </div>
                
                <div v-else class="p-4 bg-base-200/50 border border-primary/30 mb-2">
                    <div class="flex items-center gap-2 text-primary font-mono text-xs mb-2">
                        <i class="ri-check-line"></i> 日历订阅已启用
                    </div>
                    <a :href="calendarUrl" class="text-xs text-base-content/60 break-all hover:text-primary">
                        {{ calendarUrl }}
                    </a>
                    <p class="text-[10px] text-base-content/40 mt-2">点击上方链接可重新订阅。日历刷新频率由系统决定。</p>
                </div>
            </div>

            <div class="text-center opacity-50 mt-10">
                <p class="font-mono text-[10px] tracking-widest">VERSION 2.0.1_BETA</p>
            </div>
        </div>

        <!-- DB Picker Modal -->
        <dialog id="db_picker_modal" class="modal modal-bottom sm:modal-middle" :class="{ 'modal-open': showDbPicker }">
            <div class="modal-box bg-base-100 border border-primary/20 shadow-[0_0_50px_rgba(0,0,0,0.5)]">
                <div class="flex justify-between items-center mb-4">
                    <h3 class="font-bold text-lg text-primary font-display">选择默认收集箱</h3>
                    <button @click="showDbPicker = false" class="btn btn-sm btn-square btn-ghost">
                        <i class="ri-close-line"></i>
                    </button>
                </div>
                
                <div class="flex flex-col gap-2 max-h-[60vh] overflow-y-auto">
                    <button v-for="db in databases" :key="db.id" 
                        @click="selectDb(db.id)"
                        class="btn btn-outline justify-start font-normal normal-case border-base-content/10 hover:border-primary hover:bg-primary/5 hover:text-primary"
                        :class="{ 'border-primary bg-primary/10 text-primary': user?.default_database_id === db.id }"
                    >
                        <i class="ri-database-2-line mr-2"></i>
                        <span class="truncate">{{ db.name }}</span>
                        <i v-if="user?.default_database_id === db.id" class="ri-check-line ml-auto"></i>
                    </button>
                    
                    <div v-if="databases.length === 0" class="text-center py-8 text-base-content/40 font-mono text-xs">
                        <p class="mb-2">未找到 Database</p>
                        <p>请确保已在 Notion 中授权相关页面</p>
                    </div>
                </div>
            </div>
            <form method="dialog" class="modal-backdrop">
                <button @click="showDbPicker = false">close</button>
            </form>
        </dialog>
    </div>
  </div>
</template>
