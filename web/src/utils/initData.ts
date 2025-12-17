const STORAGE_KEY = "tg_todo_web_init_data";

const debug = (...args: unknown[]) => {
  if (import.meta.env.DEV) {
    console.debug("[init-data]", ...args);
  }
};

const preview = (value?: string | null) =>
  value ? `${value.slice(0, 60)}${value.length > 60 ? "…" : ""}` : "(empty)";

export const isValidInitData = (value?: string | null): value is string =>
  !!value && value.includes("hash=") && value.includes("auth_date=");

export const resolveInitData = (): string => {
  // 1. Telegram WebView 注入的 init data
  const webAppInitData = window.Telegram?.WebApp?.initData;
  if (isValidInitData(webAppInitData)) {
    persist(webAppInitData);
    debug("using initData from window.Telegram.WebApp");
    return webAppInitData;
  }
  if (webAppInitData) {
    debug("window.Telegram.WebApp.initData missing hash/auth_date", preview(webAppInitData));
  }

  const url = new URL(window.location.href);
  let queryInitData =
    url.searchParams.get("init_data") ?? url.searchParams.get("tgWebAppData");

  // 3. Hash 参数
  if (!isValidInitData(queryInitData) && url.hash.length > 1) {
    const hashParams = new URLSearchParams(url.hash.slice(1));
    if (hashParams.has("hash") && hashParams.has("auth_date")) {
      const allowKeys = [
        "user",
        "receiver",
        "chat",
        "chat_type",
        "chat_instance",
        "start_param",
        "can_send_after",
        "auth_date",
        "hash",
        "query_id",
      ];
      const parts: string[] = [];
      for (const key of allowKeys) {
        const val = hashParams.get(key);
        if (val) {
          parts.push(`${key}=${val}`);
        }
      }
      if (parts.length > 0) {
        queryInitData = parts.join("&");
      }
    }
    if (!isValidInitData(queryInitData)) {
      queryInitData =
        hashParams.get("init_data") ?? hashParams.get("tgWebAppData");
    }
  }

  if (isValidInitData(queryInitData)) {
    persist(queryInitData);
    debug("using initData from url/hash");
    return queryInitData;
  }

  const stored = localStorage.getItem(STORAGE_KEY);
  if (isValidInitData(stored)) {
    debug("falling back to localStorage", preview(stored));
    return stored;
  }
  if (stored) {
    debug("dropping stale initData from localStorage", preview(stored));
    localStorage.removeItem(STORAGE_KEY);
  }
  debug("initData not found, returning empty string");
  return "";
};

const persist = (value: string) => {
  if (!isValidInitData(value)) return;
  localStorage.setItem(STORAGE_KEY, value);
};

export const registerMockInitDataSetter = () => {
  if (!window.tgTodo) {
    window.tgTodo = {};
  }
  window.tgTodo.setMockInitData = (value: string) => {
    if (isValidInitData(value)) {
      persist(value);
      window.location.reload();
    } else {
      console.warn("[tg-miniapp] invalid mock init data, expect hash/auth_date fields");
    }
  };
  window.tgTodo.inspectInitData = () => {
    const telegram = window.Telegram?.WebApp?.initData ?? "";
    const stored = localStorage.getItem(STORAGE_KEY) ?? "";
    const resolved = resolveInitData();
    console.info("[tg-miniapp] init data snapshot", {
      telegram: preview(telegram),
      stored: preview(stored),
      resolved: preview(resolved),
      resolvedValid: isValidInitData(resolved),
    });
    return resolved;
  };
  window.tgTodo.clearInitData = () => {
    localStorage.removeItem(STORAGE_KEY);
    console.info("[tg-miniapp] cleared cached init data");
  };
};

export const extractStartParam = (): string | undefined =>
  new URLSearchParams(window.location.search).get("start_param") ?? undefined;
