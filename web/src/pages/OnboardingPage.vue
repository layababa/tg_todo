<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const router = useRouter()
const authStore = useAuthStore()
const currentSlide = ref(0)
const totalSlides = 3

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

const nextSlide = () => {
  if (currentSlide.value < totalSlides - 1) {
    currentSlide.value++
  } else {
    enterApp()
  }
}

const enterApp = () => {
  router.replace({ name: 'home' })
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
  <div class="grid-bg"></div>
  <div class="scan-line"></div>

  <div class="onboarding-container">
    <!-- Skip Button -->
    <div class="skip-btn-container">
      <button @click="enterApp" class="secondary-link skip-btn">
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
</template>

<style scoped>
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
</style>
