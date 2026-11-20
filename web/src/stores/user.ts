import { defineStore } from 'pinia'
import { ref } from 'vue'
import { loginApi, logoutApi, meApi, type UserInfo } from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  const user = ref<UserInfo | null>(null)

  async function login(username: string, password: string) {
    user.value = await loginApi(username, password)
    return user.value
  }

  async function fetchMe() {
    try {
      user.value = await meApi()
      return user.value
    } catch {
      user.value = null
      return null
    }
  }

  async function logout() {
    try {
      await logoutApi()
    } finally {
      user.value = null
    }
  }

  return { user, login, fetchMe, logout }
})
