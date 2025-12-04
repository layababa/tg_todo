import { createApp } from 'vue'
import { createPinia } from 'pinia'
import WebApp from '@twa-dev/sdk'

import App from './App.vue'
import router from './router'
import './styles/main.css'

try {
  WebApp.ready()
  WebApp.expand()
} catch (err) {
  console.warn('[tg-miniapp] Telegram WebApp bridge missing', err)
}

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
