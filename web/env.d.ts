/// <reference types="vite/client" />

declare module "vite/client" {
  interface ImportMetaEnv {
    readonly VITE_API_BASE_URL?: string;
    readonly VITE_TELEGRAM_BOT_NAME?: string;
  }
}

interface Window {
  Telegram?: {
    WebApp?: {
      initData?: string;
      initDataUnsafe?: Record<string, unknown>;
      ready?: () => void;
      expand?: () => void;
      switchInlineQuery?: (query: string, choose_types?: string[]) => void;
      [key: string]: any;
    };
  };
  tgTodo?: {
    setMockInitData?: (value: string) => void;
    inspectInitData?: () => string;
    clearInitData?: () => void;
  };
}
