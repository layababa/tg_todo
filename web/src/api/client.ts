import axios from 'axios'
import { resolveInitData } from '@/utils/initData'

const envBase = import.meta.env.VITE_API_BASE_URL
const baseURL =
  (envBase && envBase.replace(/\/$/, '')) ||
  (window.location.origin === 'null' ? 'http://localhost:8081' : window.location.origin)

const apiClient = axios.create({
  baseURL,
  withCredentials: true,
  timeout: 15000
})

apiClient.interceptors.request.use((config) => {
  const initData = resolveInitData()
  if (initData) {
    config.headers['X-Telegram-Init-Data'] = initData
  }
  return config
})

export default apiClient
