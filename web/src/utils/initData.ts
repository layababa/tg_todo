const STORAGE_KEY = 'tg_todo_web_init_data'

export const resolveInitData = (override?: string): string => {
  if (typeof override === 'string') {
    persist(override)
    return override
  }

  const url = new URL(window.location.href)
  const queryInitData = url.searchParams.get('init_data') ?? url.searchParams.get('tgWebAppData')
  if (queryInitData) {
    const decoded = safeDecode(queryInitData)
    persist(decoded)
    url.searchParams.delete('init_data')
    url.searchParams.delete('tgWebAppData')
    window.history.replaceState({}, '', url)
    return decoded
  }

  const webAppInitData = window.Telegram?.WebApp?.initData
  if (webAppInitData) {
    persist(webAppInitData)
    return webAppInitData
  }

  return localStorage.getItem(STORAGE_KEY) ?? ''
}

const persist = (value: string) => {
  if (!value) return
  localStorage.setItem(STORAGE_KEY, value)
}

const safeDecode = (value: string): string => {
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

export const registerMockInitDataSetter = () => {
  if (!window.tgTodo) {
    window.tgTodo = {}
  }
  window.tgTodo.setMockInitData = (value: string) => {
    if (typeof value === 'string') {
      persist(value)
      window.location.reload()
    }
  }
}

export const extractStartParam = (): string | undefined => {
  const fromUrl = new URLSearchParams(window.location.search).get('start_param') ?? undefined
  return fromUrl
}
