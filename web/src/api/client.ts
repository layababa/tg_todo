import axios from "axios";
import { resolveInitData } from "@/utils/initData";

const envBase = import.meta.env.VITE_API_BASE_URL;
let baseURL =
  (envBase && envBase.replace(/\/$/, "")) || "https://ddddapi.zcvyzest.xyz/api";

// 自动补全 /api 前缀以增强鲁棒性
if (baseURL && !baseURL.endsWith("/api")) {
  baseURL += "/api";
}

const apiClient = axios.create({
  baseURL,
  withCredentials: true,
  timeout: 15000,
});

apiClient.interceptors.request.use((config) => {
  const initData = resolveInitData();

  if (initData) {
    config.headers["X-Telegram-Init-Data"] = initData;
  } else {
    console.warn("[api-client] No initData found for request:", config.url);
  }

  // 打印调试信息，确认请求发往何处
  if (import.meta.env.DEV) {
    console.debug(`[api-client] Requesting: ${config.baseURL}${config.url}`);
  }

  return config;
});

export default apiClient;
