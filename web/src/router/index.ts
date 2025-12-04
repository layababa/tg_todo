import { createRouter, createWebHistory } from 'vue-router'

const OnboardingPage = () => import('@/pages/OnboardingPage.vue')
const HomePage = () => import('@/pages/HomePage.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/onboarding'
    },
    {
      path: '/onboarding',
      name: 'onboarding',
      component: OnboardingPage
    },
    {
      path: '/home',
      name: 'home',
      component: HomePage
    }
  ]
})

export default router
