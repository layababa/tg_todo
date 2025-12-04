import axios from 'axios'

const envBase = import.meta.env.VITE_API_BASE_URL
const baseURL =
  (envBase && envBase.replace(/\/$/, '')) ||
  (window.location.origin === 'null' ? 'http://localhost:8081' : window.location.origin)

const apiClient = axios.create({
  baseURL,
  withCredentials: true,
  timeout: 15000
})

export default apiClient
