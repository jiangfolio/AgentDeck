import { apiGet, apiPost, encryptedPost } from '@/api/http'

export type UserInfo = {
  id: number
  username: string
  display_name: string
}

/** 登录走国密加密传输。 */
export function loginApi(username: string, password: string) {
  return encryptedPost<UserInfo>('/auth/login', { username, password })
}

/** 登出 / 当前用户为明文 JSON。 */
export function logoutApi() {
  return apiPost('/auth/logout', {})
}

export function meApi() {
  return apiGet<UserInfo>('/auth/me')
}
