import { defineStore } from 'pinia'

interface AppState {
  isReady: boolean
}

export const useAppStore = defineStore('app', {
  state: (): AppState => ({
    isReady: false
  }),
  actions: {
    markReady() {
      this.isReady = true
    }
  }
})
