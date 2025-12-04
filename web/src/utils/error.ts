import type { AxiosError } from 'axios'

interface ErrorPayload {
  error?: { message?: string }
  message?: string
}

export const extractErrorMessage = (error: unknown): string => {
  if (!error) return '发生未知错误'

  if (typeof error === 'string') return error

  if ((error as Error).message) {
    const err = error as AxiosError<ErrorPayload>
    const responseMessage = err.response?.data?.error?.message || err.response?.data?.message
    if (responseMessage) {
      return responseMessage
    }
    return err.message
  }

  return '发生未知错误'
}
