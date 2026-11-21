<script setup lang="ts">
import { computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ApplicationOne,
  Code,
  Dashboard,
  Terminal,
  FolderOpen,
  HardDisk,
  Home,
  List,
  Shield,
  Shop,
  Timer,
} from '@icon-park/vue-next'

defineProps<{ collapsed: boolean }>()

const router = useRouter()
const route = useRoute()

const selectedKeys = computed(() => [String(route.name || 'Home')])

const items = [
  {
    key: 'Home',
    label: '首页',
    icon: () => h(Home, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentMarket',
    label: '应用市场',
    icon: () => h(Shop, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentMonitor',
    label: '监控',
    icon: () => h(Dashboard, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentProcess',
    label: '进程管理',
    icon: () => h(ApplicationOne, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentFirewall',
    label: '防火墙',
    icon: () => h(Shield, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentDisk',
    label: '磁盘管理',
    icon: () => h(HardDisk, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentFiles',
    label: '文件管理',
    icon: () => h(FolderOpen, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentCron',
    label: '定时任务',
    icon: () => h(Timer, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentScripts',
    label: '脚本库',
    icon: () => h(Code, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentStartup',
    label: '开机自启',
    icon: () => h(List, { theme: 'outline', size: 18 }),
  },
  {
    key: 'AgentTerminal',
    label: '终端',
    icon: () => h(Terminal, { theme: 'outline', size: 18 }),
  },
]

function onClick(info: { key: string | number }) {
  router.push({ name: String(info.key) })
}
</script>

<template>
  <div
    class="logo flex h-10 cursor-pointer items-center justify-center border-b border-white/10 px-4"
    @click="router.push({ name: 'Home' })"
  >
    <div class="truncate text-center text-base font-semibold text-white">
      {{ collapsed ? 'AD' : 'AgentDeck' }}
    </div>
  </div>
  <div class="menu">
    <a-menu
      mode="inline"
      theme="dark"
      :selected-keys="selectedKeys"
      :items="items"
      :inline-collapsed="collapsed"
      @click="onClick"
    />
  </div>
</template>
