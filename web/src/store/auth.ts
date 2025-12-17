import { defineStore } from "pinia";

import apiClient from "@/api/client";
import type { AuthStatusPayload } from "@/types/auth";
import { extractErrorMessage } from "@/utils/error";
import { registerMockInitDataSetter, resolveInitData } from "@/utils/initData";

registerMockInitDataSetter();

interface AuthState {
  initData: string;
  user: AuthStatusPayload["user"] | null;
  notionConnected: boolean;
  redirectHint: string | null;
  statusMessage: string;
  loadingStatus: boolean;
  loadingLink: boolean;
  error: string | null;
}

export const useAuthStore = defineStore("auth", {
  state: (): AuthState => ({
    initData: resolveInitData(),
    user: null,
    notionConnected: false,
    redirectHint: null,
    statusMessage: "",
    loadingStatus: false,
    loadingLink: false,
    error: null,
  }),
  getters: {
    hasInitData: (state) => Boolean(state.initData),
  },
  actions: {
    buildHeaders() {
      const headers: Record<string, string> = {
        Accept: "application/json",
      };
      if (this.initData) {
        headers["X-Telegram-Init-Data"] = this.initData;
      }
      return headers;
    },
    async fetchStatus(startParam?: string) {
      if (!this.hasInitData) {
        this.error = "缺少 Telegram init data，请从 Telegram 内打开应用。";
        return;
      }
      this.loadingStatus = true;
      this.error = null;
      try {
        const params = startParam ? { start_param: startParam } : undefined;
        const response = await apiClient.get<AuthStatusPayload>(
          "/auth/status",
          {
            headers: this.buildHeaders(),
            params,
          }
        );
        const payload = response.data;
        this.user = payload.user;
        this.notionConnected = Boolean(payload.notion_connected);
        this.redirectHint = payload.redirect_hint ?? startParam ?? null;
        this.statusMessage = payload.notion_connected
          ? "已连接 Notion，正在跳转..."
          : "请授权 Notion 以启用跨平台同步，或以游客模式体验。";
        return payload;
      } catch (error) {
        this.error = extractErrorMessage(error);
        throw error;
      } finally {
        this.loadingStatus = false;
      }
    },
    async requestNotionAuthUrl(): Promise<string> {
      if (!this.hasInitData) {
        const message = "缺少 Telegram init data，无法生成授权链接。";
        this.error = message;
        throw new Error(message);
      }
      this.loadingLink = true;
      this.error = null;
      try {
        const response = await apiClient.get<{ url: string }>(
          "/auth/notion/url",
          {
            headers: this.buildHeaders(),
          }
        );
        return response.data.url;
      } catch (error) {
        this.error = extractErrorMessage(error);
        throw error;
      } finally {
        this.loadingLink = false;
      }
    },
  },
});
