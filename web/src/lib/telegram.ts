import WebApp from '@twa-dev/sdk'

const DEBUG_INIT_DATA_KEY = 'tg_todo_debug_init_data'
let cachedInitData: string | null = null

const persistDebugInitData = (value: string) => {
  if (typeof window === 'undefined') return
  try {
    window.localStorage?.setItem(DEBUG_INIT_DATA_KEY, value)
  } catch (error) {
    // eslint-disable-next-line no-console
    console.warn('persist initData failed', error)
  }
}

export const initTelegramWebApp = () => {
  if (typeof window === 'undefined') {
    return
  }

  try {
    WebApp.ready()
    WebApp.expand()
    if (WebApp.initData) {
      cachedInitData = WebApp.initData
      persistDebugInitData(WebApp.initData)
    }
  } catch (error) {
    // eslint-disable-next-line no-console
    console.warn('Telegram WebApp init skipped', error)
  }
}

const resolveDebugInitData = () => {
  const envInitData = import.meta.env.VITE_TG_INIT_DATA ?? ''
  if (typeof window === 'undefined') return envInitData
  const stored = window.localStorage?.getItem(DEBUG_INIT_DATA_KEY) ?? ''
  return stored || envInitData
}

/**
 * 获取 Telegram WebApp initData，优先缓存 > runtime > 本地调试占位。
 */
export const getInitData = () => {
  if (cachedInitData) return cachedInitData
  if (typeof window === 'undefined') return ''
  const runtime = window.Telegram?.WebApp?.initData || ''
  const fallback = resolveDebugInitData()
  cachedInitData = runtime || fallback || ''
  if (!runtime && cachedInitData) {
    persistDebugInitData(cachedInitData)
  }
  return cachedInitData
}

export const setDebugInitData = (value: string) => {
  cachedInitData = value
  persistDebugInitData(value)
}

export const telegram = WebApp

export const mainButton = WebApp.MainButton

export const backButton = WebApp.BackButton
