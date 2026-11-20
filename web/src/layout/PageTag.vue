<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { AllApplication, Close, Home } from '@icon-park/vue-next'
import { useTabsStore } from '@/stores/tabs'
import './page-tag.css'

const router = useRouter()
const tabsStore = useTabsStore()

const tabs = computed(() => tabsStore.tabs)
const activeKey = computed(() => tabsStore.activeKey)

const tabActionMenu = {
  items: [
    { key: 'others', label: '关闭其他' },
    { key: 'all', label: '关闭所有' },
  ],
  onClick: ({ key }: { key: string | number }) => {
    if (String(key) === 'others') {
      onCloseOthers()
      return
    }
    onCloseAll()
  },
}

function onSelect(key: string) {
  tabsStore.activeKey = key
  const tab = tabsStore.tabs.find((t) => t.key === key)
  if (tab) router.push(tab.path)
}

function onClose(e: Event, key: string) {
  e.stopPropagation()
  const next = tabsStore.closeTab(key)
  if (next) router.push(next.path)
}

function onCloseOthers() {
  tabsStore.closeOthers(activeKey.value)
  router.push(tabsStore.tabs.find((t) => t.key === activeKey.value)?.path || '/home')
}

function onCloseAll() {
  const home = tabsStore.closeAll()
  router.push(home.path)
}
</script>

<template>
  <div class="page-tag-row">
    <div class="page-tag-main">
      <a-dropdown :trigger="['contextmenu']" :menu="tabActionMenu">
        <div class="page-tag">
          <div class="page-tag-list">
            <div
              v-for="tab in tabs"
              :key="tab.key"
              class="page-tag-item"
              :class="{ 'is-active': tab.key === activeKey }"
              @click="onSelect(tab.key)"
            >
              <span v-if="tab.key === activeKey" class="page-tag-item-mid" />
              <span class="page-tag-item-icon">
                <Home v-if="tab.key === 'Home'" theme="outline" :size="12" />
              </span>
              <span>{{ tab.title }}</span>
              <span
                v-if="tab.closable"
                class="page-tag-item-close"
                @click="onClose($event, tab.key)"
              >
                <Close theme="outline" :size="12" />
              </span>
            </div>
          </div>
        </div>
      </a-dropdown>
    </div>

    <a-dropdown :trigger="['click']" :menu="tabActionMenu">
      <div class="page-tag-operate" role="button" tabindex="0">
        <AllApplication theme="outline" :size="17" />
      </div>
    </a-dropdown>
  </div>
</template>
