/// <reference types="vite/client" />

declare module 'vite/client' {
  interface ImportMetaEnv {
    readonly VITE_API_BASE_URL?: string
    readonly VITE_TELEGRAM_BOT_NAME?: string
  }
}

interface Window {
  Telegram?: {
    WebApp?: {
      initData?: string
      initDataUnsafe?: Record<string, unknown>
      ready?: () => void
      expand?: () => void
      [key: string]: unknown
    }
  }
  tgTodo?: {
    setMockInitData?: (value: string) => void
  }
}
