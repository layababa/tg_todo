import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { initTelegramWebApp } from './lib/telegram'

import './styles/main.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

initTelegramWebApp()

app.mount('#app')
