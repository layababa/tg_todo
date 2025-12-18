<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listGroups, bindGroup, validateGroupDatabase, initGroupDatabase, unbindGroup } from '@/api/group'
import { listDatabases } from '@/api/notion'
import type { Group, DatabaseSummary, ValidationResult } from '@/types/group'

const route = useRoute()
const router = useRouter()
const groupID = route.params.groupID as string

const loading = ref(true)
const error = ref('')
const group = ref<Group | null>(null)
const databases = ref<DatabaseSummary[]>([])
const selectedDB = ref('')
const validating = ref(false)
const validationResult = ref<ValidationResult | null>(null)
const binding = ref(false)
const initing = ref(false)

const canBind = computed(() => {
  return selectedDB.value && validationResult.value?.valid
})

const fetchData = async () => {
  try {
    loading.value = true
    const [groups, dbs] = await Promise.all([listGroups(), listDatabases()])
    
    const found = groups.find(g => g.id.toString() === groupID)
    if (!found) {
      error.value = 'Group not found or you do not have permission.'
      return
    }
    group.value = found
    databases.value = dbs
    
    // If already bound, select it
    if (found.db) {
       // found.db is Summary, we need ID
       selectedDB.value = found.db.id
    }
  } catch (e: any) {
    error.value = e.message || 'Failed to load data'
  } finally {
    loading.value = false
  }
}

const onDBSelect = async () => {
  if (!selectedDB.value) {
    validationResult.value = null
    return
  }
  try {
    validating.value = true
    validationResult.value = await validateGroupDatabase(groupID, selectedDB.value)
  } catch (e) {
    console.error(e)
  } finally {
    validating.value = false
  }
}

const onBind = async () => {
  if (!canBind.value) return
  try {
    binding.value = true
    await bindGroup(groupID, selectedDB.value)
    alert('Bound successfully!')
    router.push('/home') // Or stay
  } catch (e: any) {
    alert('Bind failed: ' + e.message)
  } finally {
    binding.value = false
  }
}

const onInit = async () => {
  try {
    initing.value = true
    await initGroupDatabase(groupID, selectedDB.value)
    // Re-validate
    await onDBSelect()
  } catch (e: any) {
    alert('Init failed: ' + e.message)
  } finally {
    initing.value = false
  }
}

const onUnbind = async () => {
  if (!confirm('Are you sure you want to unbind Notion? Tasks will no longer sync.')) return
  try {
    binding.value = true
    await unbindGroup(groupID)
    alert('Unbound successfully!')
    router.push('/home')
  } catch(e: any) {
    alert('Unbind failed: ' + e.message)
  } finally {
    binding.value = false
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="min-h-screen bg-base-200 p-4">
    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center mt-20 gap-4">
      <span class="loading loading-spinner loading-lg"></span>
      <p class="text-base-content/60">Loading group info...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="alert alert-error max-w-md mx-auto mt-10">
      <i class="ri-error-warning-line text-xl"></i>
      <span>{{ error }}</span>
    </div>

    <!-- Content -->
    <div v-else class="card bg-base-100 shadow-xl max-w-md mx-auto">
      <div class="card-body">
        <h2 class="card-title text-2xl mb-4">Configure Group</h2>
        
        <div class="form-control w-full">
          <label class="label">
            <span class="label-text">Group Name</span>
          </label>
          <div class="text-lg font-medium px-1">{{ group?.title }}</div>
        </div>

        <div class="form-control w-full mt-4">
          <label class="label">
            <span class="label-text">Select Notion Database</span>
          </label>
          <select 
            v-model="selectedDB" 
            @change="onDBSelect"
            class="select select-bordered w-full"
          >
            <option value="" disabled>Select a database...</option>
            <option v-for="db in databases" :key="db.id" :value="db.id">
              {{ db.name }}
            </option>
          </select>
        </div>

        <!-- Validation Status -->
        <div v-if="selectedDB && validating" class="flex items-center gap-2 mt-4 text-sm text-base-content/60">
           <span class="loading loading-spinner loading-xs"></span> Checking compatibility...
        </div>

        <div v-if="validationResult" class="alert mt-4" :class="validationResult.valid ? 'alert-success' : 'alert-warning'">
          <i :class="validationResult.valid ? 'ri-checkbox-circle-line' : 'ri-alert-line'"></i>
          <div>
            <div class="font-bold">{{ validationResult.valid ? 'Compatible' : 'Action Needed' }}</div>
            <div v-if="!validationResult.valid" class="text-xs mt-1">
               Missing properties: {{ validationResult.missing_properties.join(', ') }}
            </div>
          </div>
          <div v-if="!validationResult.valid">
             <button class="btn btn-sm btn-outline" @click="onInit" :disabled="initing">
                {{ initing ? 'Fixing...' : 'Fix' }}
             </button>
          </div>
        </div>

        <!-- Bind Button -->
        <div class="card-actions justify-end mt-8">
           <button 
             class="btn btn-primary w-full" 
             @click="onBind" 
             :disabled="!canBind || binding"
           >
             <span v-if="binding" class="loading loading-spinner"></span>
             {{ binding ? 'Saving...' : 'Save & Bind' }}
           </button>
        </div>

        <!-- Unbind Section -->
        <div v-if="group?.db" class="divider mt-8 text-xs text-base-content/40">Danger Zone</div>
        
        <div v-if="group?.db" class="form-control">
           <button 
             class="btn btn-outline btn-error w-full" 
             @click="onUnbind" 
             :disabled="binding"
           >
             <i class="ri-link-unlink-m"></i>
             {{ binding ? 'Processing...' : 'Unbind & Disconnect Notion' }}
           </button>
           <label class="label">
             <span class="label-text-alt text-center w-full">Tasks will only be saved locally.</span>
           </label>
        </div>

      </div>
    </div>
  </div>
</template>
