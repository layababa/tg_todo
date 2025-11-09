import { createRouter, createWebHistory } from 'vue-router'

import TodoDetailView from '@/views/TodoDetailView.vue'
import TodoListView from '@/views/TodoListView.vue'

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

export default router
