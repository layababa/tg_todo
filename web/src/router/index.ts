import { createRouter, createWebHistory } from 'vue-router'

import TodoDetailView from '@/views/TodoDetailView.vue'
import TodoListView from '@/views/TodoListView.vue'
import { backButton } from '@/lib/telegram'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'todos',
      component: TodoListView
    },
    {
      path: '/todos/:id',
      name: 'todo-detail',
      component: TodoDetailView,
      props: true
    }
  ]
})

const handleTelegramBack = () => {
  if (typeof window === 'undefined') {
    router.back()
    return
  }
  if (window.history.length > 1) {
    router.back()
  } else {
    window.Telegram?.WebApp?.close()
  }
}

const toggleBackButton = (shouldShow: boolean) => {
  if (typeof window === 'undefined') return
  try {
    if (shouldShow) {
      backButton?.onClick?.(handleTelegramBack)
      backButton?.show?.()
    } else {
      backButton?.offClick?.(handleTelegramBack)
      backButton?.hide?.()
    }
  } catch (error) {
    // eslint-disable-next-line no-console
    console.warn('telegram back button toggle skipped', error)
  }
}

router.afterEach(to => {
  toggleBackButton(to.name !== 'todos')
})

export default router
