<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const router = useRouter()
const authStore = useAuthStore()
const currentSlide = ref(0)
const totalSlides = 3
const isInitializing = ref(true)
const showCarousel = ref(false)

const slides = [
  {
    icon: 'ri-flashlight-fill',
    title: '秒级创建',
    desc: '无需离开对话，在聊天中直接生成任务。'
  },
  {
    icon: 'ri-brain-line',
    title: '智能上下文',
    desc: 'AI 自动抓取讨论背景，拒绝信息丢失。'
  },
  {
    icon: 'ri-flow-chart',
    title: '灵活协作',
    desc: '在 Mini App 聚合管理，Notion 同步按需开启。'
  }
]

onMounted(async () => {
  // === Debug: Active Safe Area Request ===
  const debugSafeArea = () => {
    try {
      const WebApp = (window as any).Telegram?.WebApp
      const WebView = (window as any).Telegram?.WebView
      
      console.log('[Onboarding] Checking Safe Area...', {
        currentInset: WebApp?.safeAreaInset,
        currentContent: WebApp?.contentSafeAreaInset
      })

      // Listener for debug
      const logChange = (event: string) => {
        console.log(`[Onboarding] ${event} triggered!`, {
          newInset: WebApp?.safeAreaInset,
          newContent: WebApp?.contentSafeAreaInset
        })
      }

      WebApp?.onEvent?.('safeAreaChanged', () => logChange('safeAreaChanged'))
      WebApp?.onEvent?.('contentSafeAreaChanged', () => logChange('contentSafeAreaChanged'))

      // Active Request
      if (WebView?.postEvent) {
        console.log('[Onboarding] Sending active request for safe area...')
        WebView.postEvent('web_app_request_safe_area', {})
        WebView.postEvent('web_app_request_content_safe_area', {})
      } else {
        console.warn('[Onboarding] WebView.postEvent not available')
      }
    } catch (e) {
      console.error('[Onboarding] Failed to debug safe area', e)
    }
  }
  
  // Run debug logic immediately
  debugSafeArea()

  // Start initializing
  const startTime = Date.now()
  
  // 1. 获取 start_param (处理多种获取方式)
  let startParam = ''
  try {
    const urlParams = new URLSearchParams(window.location.search)
    // 兼容 Telegram 各种注入方式
    startParam = urlParams.get('tgWebAppStartParam') || 
                 urlParams.get('start_param') || 
                 // @ts-ignore
                 (window.Telegram?.WebApp?.initDataUnsafe?.start_param as string) || ''
  } catch (e) {
    console.warn('Failed to get start_param', e)
  }

  try {
    await authStore.fetchStatus(startParam)
    
    // Check if user has seen onboarding before
    const hasCompletedOnboarding = localStorage.getItem('onboarding_completed') === 'true'
    
    // Minimum splash time
    const minSplashTime = 1200 
    const elapsedTime = Date.now() - startTime
    const remainingTime = Math.max(0, minSplashTime - elapsedTime)
    
    setTimeout(() => {
      // 只要有 start_param，就直接尝试进入 App 进行跳转
      if (startParam || (hasCompletedOnboarding && authStore.user)) {
        enterApp()
      } else {
        isInitializing.value = false
        showCarousel.value = true
      }
    }, remainingTime)
    
  } catch (e) {
    console.error('Initialization failed', e)
    isInitializing.value = false
    showCarousel.value = true
  }
})

const nextSlide = () => {
  if (currentSlide.value < totalSlides - 1) {
    currentSlide.value++
  } else {
    completeOnboarding()
  }
}

const completeOnboarding = () => {
  localStorage.setItem('onboarding_completed', 'true')
  enterApp()
}

const enterApp = () => {
  const hint = authStore.redirectHint
  if (hint && hint.startsWith('task_')) {
    const taskId = hint.replace('task_', '')
    router.replace({ name: 'task-detail', params: { id: taskId } })
  } else {
    router.replace({ name: 'home' })
  }
}

const connectNotion = async () => {
  try {
    const url = await authStore.requestNotionAuthUrl()
    window.location.href = url
  } catch (e) {
    console.error('Failed to get auth url', e)
    // Fallback or error toast could go here
    // For now, if it fails, maybe just let them enter app or show alert
    alert('连接服务暂不可用，请稍后重试')
  }
}
</script>

<template>
  <div class="page-root">


    <!-- Splash Screen / Startup Animation -->
    <transition name="fade">
      <div v-if="isInitializing" class="splash-container">
        <div class="splash-logo">
          <div class="logo-outer"></div>
          <div class="logo-inner">
            <i class="ri-flashlight-fill"></i>
          </div>
        </div>
        <div class="splash-text">
          <span class="glitch-text" data-text="TG TODO">TG TODO</span>
          <div class="loading-bar-container">
            <div class="loading-bar"></div>
          </div>
          <div class="status-text">INITIALIZING SYSTEM...</div>
        </div>
      </div>
    </transition>

    <!-- Onboarding Carousel -->
    <div v-if="showCarousel" class="onboarding-container">
      <!-- Skip Button -->
      <div class="skip-btn-container">
        <button @click="completeOnboarding" class="secondary-link skip-btn">
          SKIP
        </button>
      </div>

      <!-- Carousel Area -->
      <div class="carousel-content">
        <transition name="fade-slide" mode="out-in">
          <div :key="currentSlide" class="slide-item">
            <div class="hero-graphic">
              <i :class="slides[currentSlide].icon"></i>
            </div>
            
            <h2 class="onboarding-title">{{ slides[currentSlide].title }}</h2>
            <p class="onboarding-subtitle">{{ slides[currentSlide].desc }}</p>
          </div>
        </transition>
      </div>

      <!-- Indicators -->
      <div class="indicators">
        <div 
          v-for="i in totalSlides" 
          :key="i"
          class="indicator-dot"
          :class="{ active: (i - 1) === currentSlide }"
        ></div>
      </div>

      <!-- Action Area -->
      <div class="action-area">
        <button 
          @click="nextSlide" 
          class="primary-btn"
        >
          {{ currentSlide === totalSlides - 1 ? '立即体验' : '下一步' }}
        </button>

        <button 
           v-if="currentSlide === totalSlides - 1"
           @click="connectNotion"
           class="secondary-link flex-center"
        >
          <i class="ri-notion-fill" style="margin-right: 4px;"></i> (可选) 连接 Notion
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.splash-container {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    z-index: 100;
    background: var(--bg-color, #000000);
}

.splash-logo {
    position: relative;
    width: 80px;
    height: 80px;
    margin-bottom: 40px;
}

.logo-outer {
    position: absolute;
    top: -10%;
    left: -10%;
    width: 120%;
    height: 120%;
    border: 2px solid var(--neon-green, #ABF600);
    border-radius: 35% 65% 70% 30% / 30% 30% 70% 70%;
    animation: morph 3s ease-in-out infinite;
    opacity: 0.3;
}

.logo-inner {
    width: 100%;
    height: 100%;
    background: rgba(171, 246, 0, 0.1);
    border: 1px solid var(--neon-green, #ABF600);
    display: flex;
    align-items: center;
    justify-content: center;
    clip-path: polygon(20% 0%, 80% 0%, 100% 20%, 100% 80%, 80% 100%, 20% 100%, 0% 80%, 0% 20%);
}

.logo-inner i {
    font-size: 32px;
    color: var(--neon-green, #ABF600);
    text-shadow: 0 0 10px rgba(171, 246, 0, 0.5);
}

.splash-text {
    text-align: center;
}

.glitch-text {
    font-family: var(--font-mono, monospace);
    font-size: 24px;
    font-weight: 700;
    letter-spacing: 4px;
    color: #FFFFFF;
    margin-bottom: 20px;
    display: block;
}

.loading-bar-container {
    width: 200px;
    height: 2px;
    background: rgba(255, 255, 255, 0.1);
    margin: 20px auto;
    overflow: hidden;
}

.loading-bar {
    width: 100%;
    height: 100%;
    background: var(--neon-green, #ABF600);
    animation: loading 1.5s ease-in-out infinite;
    transform-origin: left;
}

.status-text {
    font-family: var(--font-mono, monospace);
    font-size: 10px;
    color: var(--text-secondary, #666666);
    letter-spacing: 2px;
}

@keyframes morph {
    0% { border-radius: 35% 65% 70% 30% / 30% 30% 70% 70%; }
    50% { border-radius: 50% 50% 33% 67% / 55% 27% 73% 45%; }
    100% { border-radius: 35% 65% 70% 30% / 30% 30% 70% 70%; }
}

@keyframes loading {
    0% { transform: translateX(-100%) scaleX(0.1); }
    50% { transform: translateX(0) scaleX(0.5); }
    100% { transform: translateX(100%) scaleX(0.1); }
}

.onboarding-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    padding: 40px;
    text-align: center;
    position: relative;
    overflow: hidden;
}

.skip-btn-container {
    position: absolute;
    top: 24px;
    right: 24px;
    z-index: 20;
}

.skip-btn {
    font-size: 12px;
    opacity: 0.6;
}

.carousel-content {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
}

.slide-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    max-width: 400px;
}

.hero-graphic {
    width: 120px;
    height: 120px;
    border: 1px solid var(--neon-green, #ABF600);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 40px;
    box-shadow: 0 0 30px rgba(171, 246, 0, 0.1);
    position: relative;
}

.hero-graphic i {
    font-size: 48px;
    color: var(--neon-green, #ABF600);
}

.hero-graphic::after {
    content: '';
    position: absolute;
    width: 140%;
    height: 140%;
    border: 1px dashed var(--border-dim, #333333);
    border-radius: 50%;
    animation: spin 10s linear infinite;
}

.onboarding-title {
    font-size: 32px;
    font-weight: 300;
    margin-bottom: 16px;
    letter-spacing: -1px;
    color: var(--text-primary, #FFFFFF);
}

.onboarding-subtitle {
    color: var(--text-secondary, #666666);
    margin-bottom: 20px;
    font-family: var(--font-mono, monospace);
    font-size: 14px;
    line-height: 1.6;
}

.primary-btn {
    width: 100%;
    padding: 16px;
    background: var(--neon-green, #ABF600);
    color: #000000;
    border: none;
    font-family: var(--font-mono, monospace);
    font-weight: 700;
    font-size: 14px;
    cursor: pointer;
    clip-path: polygon(10px 0, 100% 0, 100% calc(100% - 10px), calc(100% - 10px) 100%, 0 100%, 0 10px);
    transition: transform 0.2s, background 0.2s;
}

.primary-btn:active {
    transform: scale(0.98);
}

.primary-btn:hover {
    background: #c2ff1f;
}

.secondary-link {
    margin-top: 16px;
    color: var(--text-secondary, #666666);
    font-size: 12px;
    text-decoration: none;
    border: none;
    background: transparent;
    cursor: pointer;
    transition: color 0.2s;
}

.secondary-link:hover {
    color: var(--text-primary, #FFFFFF);
}

.indicators {
    display: flex;
    gap: 8px;
    margin-bottom: 32px;
}

.indicator-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--border-dim, #333333);
    transition: all 0.3s ease;
}

.indicator-dot.active {
    background: var(--neon-green, #ABF600);
    width: 24px;
    border-radius: 4px;
}

.action-area {
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.flex-center {
    display: flex;
    align-items: center;
    justify-content: center;
}

@keyframes spin {
    100% {
        transform: rotate(360deg);
    }
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.3s ease;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
