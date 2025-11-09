import WebApp from '@twa-dev/sdk'

export const initTelegramWebApp = () => {
  if (typeof window === 'undefined') {
    return
  }

  try {
    WebApp.ready()
    WebApp.expand()
  } catch (error) {
    // eslint-disable-next-line no-console
    console.warn('Telegram WebApp init skipped', error)
  }
}

export const telegram = WebApp

export const mainButton = WebApp.MainButton
