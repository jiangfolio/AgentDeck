<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Info, MenuFoldOne, MenuUnfoldOne, Home as HomeIcon, Logout } from '@icon-park/vue-next'
import { message } from 'antdv-next'
import SideMenu from './SideMenu.vue'
import PageTag from './PageTag.vue'
import { useUserStore } from '@/stores/user'
import { useTabsStore } from '@/stores/tabs'

const collapsed = ref(false)
const aboutOpen = ref(false)
const repoUrl = 'https://github.com/jiangfolio/AgentDeck'
const router = useRouter()
const userStore = useUserStore()
const tabsStore = useTabsStore()

function goHome() {
  router.push({ name: 'Home' })
  tabsStore.openTab({
    key: 'Home',
    title: '首页',
    path: '/home',
    closable: false,
  })
}

async function onLogout() {
  await userStore.logout()
  tabsStore.closeAll()
  message.success('已退出登录')
  router.replace({ name: 'Login' })
}
</script>

<template>
  <a-layout class="agentdeck-layout !h-screen overflow-hidden">
    <a-layout-sider
      v-model:collapsed="collapsed"
      collapsible
      :trigger="null"
      :width="200"
      theme="dark"
      class="!h-screen overflow-y-auto"
    >
      <SideMenu :collapsed="collapsed" />
    </a-layout-sider>

    <a-layout class="h-screen overflow-hidden" style="background: var(--ad-color-bg-layout)">
      <a-layout-header
        class="!flex !h-10 !shrink-0 !items-center !justify-between !px-3"
        style="background: var(--ad-color-bg-container); border-bottom: 1px solid var(--ad-color-border-secondary)"
      >
        <div class="flex items-center gap-3">
          <span class="cursor-pointer" @click="collapsed = !collapsed">
            <MenuUnfoldOne v-if="collapsed" theme="outline" :size="20" />
            <MenuFoldOne v-else theme="outline" :size="20" />
          </span>
          <span class="cursor-pointer" title="首页" @click="goHome">
            <HomeIcon theme="outline" :size="20" />
          </span>
        </div>
        <div class="flex items-center gap-3 text-sm">
          <span
            class="flex cursor-pointer items-center gap-1"
            title="关于"
            @click="aboutOpen = true"
          >
            <Info theme="outline" :size="16" />
            关于
          </span>
          <span>{{ userStore.user?.display_name || userStore.user?.username }}</span>
          <span
            class="flex cursor-pointer items-center gap-1"
            style="color: var(--ad-color-primary)"
            @click="onLogout"
          >
            <Logout theme="outline" :size="16" />
            退出
          </span>
        </div>
      </a-layout-header>

      <PageTag class="!shrink-0" />

      <a-modal v-model:open="aboutOpen" title="关于 AgentDeck" :footer="null" centered width="480px">
        <p class="about-lead">
          AgentDeck 是跑在自己机器上的基础设施管理面板。看主机资源、管进程和计划任务、改防火墙和磁盘、浏览文件，也能开本机或远程终端。
        </p>
        <p>作者：jiangbyte</p>
        <p>
          仓库：
          <a :href="repoUrl" target="_blank" rel="noreferrer">github.com/jiangfolio/AgentDeck</a>
        </p>
        <p class="about-meta">Apache License 2.0</p>
      </a-modal>

      <a-layout-content class="ad-layout-content">
        <div class="ad-layout-content-inner">
          <router-view v-slot="{ Component }">
            <keep-alive>
              <component :is="Component" />
            </keep-alive>
          </router-view>
        </div>
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.about-lead {
  margin-bottom: 12px;
  line-height: 1.6;
}
.about-meta {
  margin-top: 8px;
  color: var(--ad-color-text-secondary);
}
</style>
