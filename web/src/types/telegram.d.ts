/**
 * Telegram Mini App SDK type definitions
 */

interface TelegramWebApp {
  initData: string
  initDataUnsafe: {
    query_id?: string
    user?: {
      id: number
      is_bot: boolean
      first_name: string
      last_name?: string
      username?: string
      language_code?: string
      is_premium?: boolean
      added_to_attachment_menu?: boolean
    }
    auth_date: number
    hash: string
    [key: string]: any
  }
  version: string
  platform: string
  headerColor: string
  backgroundColor: string
  textColor: string
  hintColor: string
  isExpanded: boolean
  viewportHeight: number
  viewportStableHeight: number
  isClosingConfirmationEnabled: boolean
  bottomBarColor: string
  ready: () => void
  expand: () => void
  close: () => void
  onEvent: (eventType: string, callback: () => void) => void
  offEvent: (eventType: string, callback: () => void) => void
  sendData: (data: string) => void
  openTelegramLink: (url: string) => void
  openLink: (url: string) => void
  showPopup: (params: Record<string, any>) => void
  showAlert: (message: string) => void
  showConfirm: (message: string) => void
  [key: string]: any
}

interface Telegram {
  WebApp: TelegramWebApp
  [key: string]: any
}

declare global {
  interface Window {
    Telegram?: Telegram
  }
}

export {}

