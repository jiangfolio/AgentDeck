import { defineStore } from 'pinia'
import { ref } from 'vue'

export type TabItem = {
  key: string
  title: string
  path: string
  closable: boolean
}

const HOME: TabItem = {
  key: 'Home',
  title: '首页',
  path: '/home',
  closable: false,
}

export const useTabsStore = defineStore('tabs', () => {
  const tabs = ref<TabItem[]>([{ ...HOME }])
  const activeKey = ref(HOME.key)

  function openTab(tab: TabItem) {
    if (!tabs.value.find((t) => t.key === tab.key)) {
      tabs.value.push(tab)
    }
    activeKey.value = tab.key
  }

  function closeTab(key: string) {
    const idx = tabs.value.findIndex((t) => t.key === key)
    if (idx < 0) return
    const target = tabs.value[idx]
    if (!target.closable) return
    tabs.value.splice(idx, 1)
    if (activeKey.value === key) {
      const next = tabs.value[idx] || tabs.value[idx - 1] || HOME
      activeKey.value = next.key
      return next
    }
    return null
  }

  function closeOthers(key: string) {
    tabs.value = tabs.value.filter((t) => t.key === key || !t.closable)
    activeKey.value = key
  }

  function closeAll() {
    tabs.value = tabs.value.filter((t) => !t.closable)
    activeKey.value = HOME.key
    return HOME
  }

  return { tabs, activeKey, openTab, closeTab, closeOthers, closeAll }
})
