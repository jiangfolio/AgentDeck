<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { message } from 'antdv-next'
import TerminalPane from './terminal/TerminalPane.vue'
import TerminalHosts from './terminal/TerminalHosts.vue'
import TerminalQuickCmds from './terminal/TerminalQuickCmds.vue'
import TerminalSettings from './terminal/TerminalSettings.vue'

const tab = ref('term')
const paneRef = ref<InstanceType<typeof TerminalPane> | null>(null)
const navItems = [
  { key: 'term', label: '终端' },
  { key: 'hosts', label: '主机' },
  { key: 'cmds', label: '快速命令' },
  { key: 'settings', label: '配置' },
]

function switchTerminal() {
  tab.value = 'term'
}

watch(tab, (key) => {
  if (key !== 'term') return
  void nextTick(() => paneRef.value?.refit())
})

function onInject(cmd: string) {
  tab.value = 'term'
  void Promise.resolve().then(() => {
    const ok = paneRef.value?.injectToActive(cmd)
    if (!ok) message.warning('请先打开并保持一个终端会话')
  })
}
</script>

<template>
  <div class="term-page">
    <div class="term-nav">
      <button
        v-for="item in navItems"
        :key="item.key"
        type="button"
        class="term-nav-item"
        :class="{ active: tab === item.key }"
        @click="tab = item.key"
      >
        {{ item.label }}
      </button>
    </div>
    <!-- 内容区固定高度，子页用 ad-page-fill 把分页贴底 -->
    <div class="term-body">
      <TerminalPane v-show="tab === 'term'" ref="paneRef" class="term-panel" />
      <TerminalHosts v-show="tab === 'hosts'" class="term-panel" @switch-terminal="switchTerminal" />
      <TerminalQuickCmds
        v-show="tab === 'cmds'"
        class="term-panel"
        @inject="onInject"
        @switch-terminal="switchTerminal"
      />
      <TerminalSettings v-show="tab === 'settings'" class="term-panel" />
    </div>
  </div>
</template>

<style scoped>
.term-page {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--ad-color-bg-container);
  padding: 0 16px 12px;
  box-sizing: border-box;
}

.term-nav {
  flex-shrink: 0;
  display: flex;
  gap: 4px;
  border-bottom: 1px solid var(--ad-color-border-secondary);
  margin-bottom: 12px;
}

.term-nav-item {
  position: relative;
  padding: 10px 16px;
  border: none;
  background: transparent;
  color: var(--ad-color-text);
  font-size: 14px;
  cursor: pointer;
}

.term-nav-item.active {
  color: var(--ad-color-primary);
  font-weight: 500;
}

.term-nav-item.active::after {
  content: '';
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: -1px;
  height: 2px;
  background: var(--ad-color-primary);
}

.term-body {
  flex: 1;
  min-height: 0;
  height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.term-panel {
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}
</style>
