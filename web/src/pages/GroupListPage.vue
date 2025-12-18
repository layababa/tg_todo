<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { listGroups, unbindGroup } from '@/api/group'
import type { Group } from '@/types/group'

const router = useRouter()
const groups = ref<Group[]>([])
const loading = ref(true)
const processingId = ref<string | null>(null)

const fetchGroups = async () => {
    loading.value = true
    try {
        const res = await listGroups()
        groups.value = res
    } catch (e) {
        console.error('Failed to fetch groups', e)
    } finally {
        loading.value = false
    }
}

const handleUnbind = async (id: string, title: string) => {
    if (!confirm(`确定要解除 "${title}" 的数据库绑定吗？\n解绑后任务不会删除，但新消息将不再同步。`)) return

    processingId.value = id
    try {
        await unbindGroup(id)
        // Refresh local state
        const g = groups.value.find(x => x.id === id)
        if (g) {
            g.status = 'Unbound'
            g.db = undefined
        }
    } catch (e) {
        alert('解绑失败')
        console.error(e)
    } finally {
        processingId.value = null
    }
}

const goBack = () => router.push('/settings')

onMounted(fetchGroups)
</script>

<template>
  <div class="grid-bg"></div>
  <div class="scan-line"></div>

  <div class="app-container">
      <!-- Header -->
      <header class="header sticky top-0 z-30">
          <div class="flex items-center justify-between mb-6">
              <button @click="goBack" class="icon-btn tech-btn !w-auto !px-2 !border-none text-sm gap-2">
                  <i class="ri-arrow-left-line"></i> 返回设置
              </button>
              <div class="font-mono text-[10px] text-primary border border-primary px-1.5 py-0.5 tracking-widest">GROUPS</div>
          </div>
          <h1 class="text-2xl font-light mb-2">群组连接管理</h1>
          <p class="text-xs text-base-content/60 font-mono">管理 Telegram 群组与 Notion Database 的绑定关系</p>
      </header>

      <main class="px-5 pb-24">
          <div v-if="loading" class="flex justify-center mt-20">
              <span class="loading loading-spinner text-primary"></span>
          </div>

          <div v-else-if="groups.length === 0" class="text-center py-20 text-base-content/40">
              <i class="ri-group-line text-4xl mb-4 block"></i>
              <p class="text-sm">暂无关联群组</p>
              <p class="text-xs mt-2">将 Bot 邀请进群组即可开始</p>
          </div>

          <div v-else class="flex flex-col gap-4">
              <div v-for="group in groups" :key="group.id" 
                  class="bg-base-200/50 border border-base-content/10 rounded-lg p-4 relative overflow-hidden transition-all hover:border-base-content/30"
              >
                  <div class="flex justify-between items-start mb-3">
                      <div>
                          <div class="font-bold text-lg mb-1">{{ group.title }}</div>
                          <div class="text-[10px] font-mono text-base-content/40">ID: {{ group.id }}</div>
                      </div>
                      <div class="badge badge-sm font-mono" 
                          :class="group.status === 'Connected' ? 'badge-primary text-black' : 'badge-ghost opacity-50'">
                          {{ group.status }}
                      </div>
                  </div>

                  <div v-if="group.status === 'Connected' && group.db" class="bg-base-300/50 rounded p-3 mb-4 flex items-center gap-3">
                      <i class="ri-database-2-fill text-primary"></i>
                      <div class="overflow-hidden">
                          <div class="text-[10px] uppercase font-mono opacity-50 mb-0.5">Linked Database</div>
                          <div class="text-sm font-mono truncate">{{ group.db.name }}</div>
                      </div>
                  </div>
                  <div v-else class="bg-base-300/30 rounded p-3 mb-4 text-xs italic opacity-50 flex items-center gap-2">
                       <i class="ri-link-unlink-m"></i> 未绑定数据库
                  </div>

                  <div class="flex justify-end gap-2 border-t border-base-content/5 pt-3">
                      <button 
                          v-if="group.status === 'Connected'"
                          @click="handleUnbind(group.id, group.title)"
                          :disabled="!!processingId"
                          class="btn btn-xs btn-outline btn-error font-mono"
                      >
                          <span v-if="processingId === group.id" class="loading loading-spinner loading-xs"></span>
                          <span v-else>UNBIND</span>
                      </button>
                      <button v-else class="btn btn-xs btn-ghost font-mono opacity-50 cursor-not-allowed">
                          UNBOUND
                      </button>
                  </div>
              </div>
          </div>
      </main>
  </div>
</template>
