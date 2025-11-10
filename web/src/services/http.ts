import axios from 'axios'

import { getInitData } from '@/lib/telegram'

const DEFAULT_API_BASE = 'https://api.xwqpfzmlj.com'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? DEFAULT_API_BASE,
  timeout: 10_000
})

http.interceptors.request.use(config => {
  const initData = getInitData()
  if (initData) {
    config.headers = {
      ...(config.headers ?? {}),
      'X-Telegram-Init-Data': initData
    }
  }
  return config
})

export default http
