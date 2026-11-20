<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'antdv-next'
import { useUserStore } from '@/stores/user'
import loginGif from '@/assets/login/login-min.gif'
import './login.css'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const loading = ref(false)
const formRef = ref()

const loginForm = reactive({
  username: '',
  password: '',
})

const rules = {
  username: [{ required: true, message: '用户名不能为空' }],
  password: [{ required: true, message: '密码不能为空' }],
}

onMounted(() => {
  document.onkeyup = (e) => {
    if (e.key === 'Enter') {
      onLogin()
    }
  }
})

onUnmounted(() => {
  document.onkeyup = null
})

async function onLogin() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    await userStore.login(loginForm.username, loginForm.password)
    message.success('登录成功')
    const redirect = (route.query.redirect as string) || '/home'
    await router.replace(redirect)
  } catch (e: any) {
    message.error(e?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <div class="box-item desc">
      <div class="welcome">
        <p>欢迎登录 AgentDeck</p>
      </div>
      <img class="welcome-img" :src="loginGif" alt="welcome" />
    </div>

    <div class="box-item login">
      <div class="login-title">账号登录</div>
      <a-form ref="formRef" class="login-form" :model="loginForm" :rules="rules" autocomplete="off">
        <a-form-item name="username">
          <a-input
            v-model:value.trim="loginForm.username"
            autocomplete="off"
            placeholder="请输入用户名"
          />
        </a-form-item>
        <a-form-item name="password">
          <a-input-password
            v-model:value="loginForm.password"
            autocomplete="new-password"
            placeholder="请输入密码"
          />
        </a-form-item>
        <a-form-item>
          <div class="btn" :class="{ 'is-loading': loading }" @click="onLogin">
            {{ loading ? '登录中...' : '登录' }}
          </div>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>
