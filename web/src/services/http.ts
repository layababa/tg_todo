import axios from 'axios'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  timeout: 10_000
})

http.interceptors.request.use(config => {
  if (typeof window !== 'undefined' && window.Telegram) {
    const initData = window.Telegram.WebApp?.initData || ''
    if (initData) {
      config.headers['X-Telegram-Init-Data'] = initData
    }
  }
  return config
})

export default http
