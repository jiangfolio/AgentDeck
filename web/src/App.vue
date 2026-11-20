<script setup lang="ts">
import { theme } from 'antdv-next'
import zhCN from 'antdv-next/locale/zh_CN'

/** 直角与主色走 antdv 主题 token，不在全局 CSS 强写 border-radius */
const appTheme = {
  token: {
    colorPrimary: '#1677ff',
    borderRadius: 0,
  },
  algorithm: theme.defaultAlgorithm,
}
</script>

<template>
  <a-config-provider :locale="zhCN" :theme="appTheme">
    <a-app class="h-full">
      <router-view />
    </a-app>
  </a-config-provider>
</template>
