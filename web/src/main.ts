import { createApp } from 'vue'
import { createPinia } from 'pinia'
import WebApp from '@twa-dev/sdk'

import App from './App.vue'
import router from './router'
import './styles/main.css'
import { useAuthStore } from '@/store/auth'

try {
  WebApp.ready()
  WebApp.expand()
} catch (err) {
  console.warn('[tg-miniapp] Telegram WebApp bridge missing', err)
}

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

if (import.meta.env.DEV) {
  const authStore = useAuthStore(pinia)
  // @ts-expect-error expose for debugging
  window.authStore = authStore
  console.debug('[main] authStore mounted on window.authStore')
}

app.mount('#app')
