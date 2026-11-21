<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { message } from 'antdv-next'
import { FullScreen, Plus } from '@icon-park/vue-next'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { terminalWsUrl, termHostListApi, type TermHost } from '@/api/agent'
import { useTerminalBridge, type OpenSessionRequest } from '@/composables/useTerminalBridge'
import {
  darkTheme,
  lightTheme,
  terminalSettings,
} from '@/composables/useTerminalSettings'

type Sess = {
  id: string
  title: string
  kind: 'local' | 'ssh'
  hostId?: string
  term: Terminal
  fit: FitAddon
  ws: WebSocket | null
  el: HTMLDivElement | null
  inject?: string
}

const { pending, takePendingSession } = useTerminalBridge()

const sessions = ref<Sess[]>([])
const activeId = ref('')
const wrapRef = ref<HTMLDivElement | null>(null)
const fullscreen = ref(false)
let seq = 0

function themeOpts() {
  return {
    fontSize: terminalSettings.fontSize,
    theme: terminalSettings.theme === 'light' ? lightTheme : darkTheme,
    cursorBlink: true,
    convertEol: true,
  }
}

/**
 * openSession 创建 xterm 实例并建立 WebSocket。
 */
function openSession(req: OpenSessionRequest) {
  // 1. 创建终端与插件
  const id = `s${++seq}`
  const fit = new FitAddon()
  const term = new Terminal(themeOpts())
  term.loadAddon(fit)
  term.loadAddon(new WebLinksAddon())
  const sess: Sess = {
    id,
    title: req.title,
    kind: req.kind,
    hostId: req.hostId,
    term,
    fit,
    ws: null,
    el: null,
    inject: req.inject,
  }
  sessions.value.push(sess)
  activeId.value = id

  // 2. 等容器有高度再 fit 并连接，避免 rows=1 把画面挤成一条
  void nextTick(() => {
    const host = document.getElementById(`term-host-${id}`) as HTMLDivElement | null
    if (!host) return
    sess.el = host
    term.open(host)
    term.onData((data) => {
      if (sess.ws && sess.ws.readyState === WebSocket.OPEN) {
        sess.ws.send(data)
      }
    })
    term.onResize(({ cols, rows }) => {
      if (sess.ws && sess.ws.readyState === WebSocket.OPEN) {
        sess.ws.send(JSON.stringify({ type: 'resize', cols, rows }))
      }
    })
    let tries = 0
    const start = () => {
      tries += 1
      try {
        fit.fit()
      } catch {
        /* 容器尚未布局 */
      }
      if ((term.rows > 2 && host.clientHeight > 40) || tries > 40) {
        connect(sess)
        return
      }
      window.setTimeout(start, 50)
    }
    start()
  })
}

/**
 * connect 按会话参数建立 WS，并在打开后可选注入脚本。
 */
function connect(sess: Sess) {
  const cols = sess.term.cols || 120
  const rows = sess.term.rows || 30
  const url = terminalWsUrl({
    kind: sess.kind,
    host_id: sess.hostId,
    cols,
    rows,
    ignore_host: terminalSettings.ignoreHostKey,
  })
  const ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'
  sess.ws = ws
  ws.onopen = () => {
    sess.fit.fit()
    ws.send(JSON.stringify({ type: 'resize', cols: sess.term.cols, rows: sess.term.rows }))
    if (sess.inject) {
      ws.send(sess.inject.endsWith('\n') ? sess.inject : `${sess.inject}\n`)
      sess.inject = undefined
    }
  }
  ws.onmessage = (ev) => {
    if (typeof ev.data === 'string') {
      sess.term.write(ev.data)
      return
    }
    sess.term.write(new Uint8Array(ev.data as ArrayBuffer))
  }
  ws.onclose = () => {
    sess.term.writeln('\r\n\x1b[90m[会话已关闭]\x1b[0m')
    sess.ws = null
  }
  ws.onerror = () => {
    sess.term.writeln('\r\n\x1b[31m[连接失败]\x1b[0m')
  }
}

function closeSession(id: string) {
  const idx = sessions.value.findIndex((s) => s.id === id)
  if (idx < 0) return
  const sess = sessions.value[idx]
  sess.ws?.close()
  sess.term.dispose()
  sessions.value.splice(idx, 1)
  if (activeId.value === id) {
    activeId.value = sessions.value[Math.max(0, idx - 1)]?.id || ''
  }
}

function injectToActive(text: string) {
  const sess = sessions.value.find((s) => s.id === activeId.value)
  if (!sess?.ws || sess.ws.readyState !== WebSocket.OPEN) return false
  const payload = text.endsWith('\n') ? text : `${text}\n`
  sess.ws.send(payload)
  return true
}

const pickerOpen = ref(false)
const hostLoading = ref(false)
const hosts = ref<TermHost[]>([])
const pickedHost = ref('local')

/**
 * openPicker 打开新建会话，并加载可选主机。
 */
async function openPicker() {
  // 1. 先弹出选择框，默认本机
  pickedHost.value = 'local'
  pickerOpen.value = true
  // 2. 拉取已保存主机，供 SSH 会话选择
  hostLoading.value = true
  try {
    const data = await termHostListApi({ page: 1, page_size: 100 })
    hosts.value = data.items || []
  } catch (e: any) {
    hosts.value = []
    message.error(e?.message || '加载主机失败')
  } finally {
    hostLoading.value = false
  }
}

/**
 * confirmNew 按所选主机打开本机或 SSH 会话。
 */
function mountToBody() {
  return document.body
}

function confirmNew() {
  // 1. 本机直接开本地 PTY
  if (pickedHost.value === 'local') {
    openSession({ kind: 'local', title: `本机 ${seq + 1}` })
    pickerOpen.value = false
    return
  }
  // 2. 已保存主机走 SSH；未选中则留在弹窗
  const host = hosts.value.find((h) => h.id === pickedHost.value)
  if (!host) {
    message.warning('请选择主机')
    return
  }
  openSession({
    kind: 'ssh',
    hostId: host.id,
    title: host.title || `${host.username}@${host.host}`,
  })
  pickerOpen.value = false
}

function onNewLocal() {
  openSession({ kind: 'local', title: `本机 ${seq + 1}` })
}

function toggleFullscreen() {
  fullscreen.value = !fullscreen.value
  void nextTick(() => fitActive())
}

function fitActive() {
  const sess = sessions.value.find((s) => s.id === activeId.value)
  if (!sess) return
  try {
    sess.fit.fit()
  } catch {
    /* ignore */
  }
}

watch(activeId, () => void nextTick(() => fitActive()))
watch(
  () => [terminalSettings.fontSize, terminalSettings.theme] as const,
  () => {
    for (const s of sessions.value) {
      s.term.options.fontSize = terminalSettings.fontSize
      s.term.options.theme = terminalSettings.theme === 'light' ? lightTheme : darkTheme
      try {
        s.fit.fit()
      } catch {
        /* ignore */
      }
    }
  },
)

watch(pending, (v) => {
  if (!v) return
  const req = takePendingSession()
  if (req) openSession(req)
})

let ro: ResizeObserver | null = null
onMounted(() => {
  if (wrapRef.value) {
    ro = new ResizeObserver(() => fitActive())
    ro.observe(wrapRef.value)
  }
  // 优先消费跨页打开请求，否则默认开本机会话
  const req = takePendingSession()
  if (req) openSession(req)
  else onNewLocal()
})

onBeforeUnmount(() => {
  ro?.disconnect()
  for (const s of [...sessions.value]) {
    closeSession(s.id)
  }
})

function refit() {
  void nextTick(() => fitActive())
}

defineExpose({ injectToActive, openSession, refit })
</script>

<template>
  <div class="pane" :class="{ 'is-full': fullscreen }" ref="wrapRef">
    <div class="bar">
      <div class="tabs">
        <button type="button" class="icon-btn" title="新建会话" @click="openPicker">
          <Plus theme="outline" :size="16" />
        </button>
        <div
          v-for="s in sessions"
          :key="s.id"
          class="tab"
          :class="{ active: s.id === activeId }"
          @click="activeId = s.id"
        >
          <span class="tab-title">{{ s.title }}</span>
          <span class="tab-close" @click.stop="closeSession(s.id)">×</span>
        </div>
      </div>
      <button type="button" class="icon-btn" title="全屏" @click="toggleFullscreen">
        <FullScreen theme="outline" :size="16" />
      </button>
    </div>
    <div class="stage">
      <div
        v-for="s in sessions"
        :id="`term-host-${s.id}`"
        :key="s.id"
        class="term-host"
        :class="{ hidden: s.id !== activeId }"
      />
      <div v-if="!sessions.length" class="empty">点击 + 选择主机并新建终端</div>
    </div>

    <a-modal
      v-model:open="pickerOpen"
      title="新建会话"
      ok-text="连接"
      centered
      :get-container="mountToBody"
      :mask-closable="true"
      destroy-on-hidden
      @ok="confirmNew"
    >
      <a-form layout="vertical">
        <a-form-item label="主机">
          <a-select
            v-model:value="pickedHost"
            show-search
            option-filter-prop="label"
            placeholder="请选择主机"
            :loading="hostLoading"
            :get-popup-container="mountToBody"
            :options="[
              { value: 'local', label: '本机' },
              ...hosts.map((h) => ({
                value: h.id,
                label: `${h.title || h.host}（${h.username}@${h.host}:${h.port}）`,
              })),
            ]"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.pane {
  display: flex;
  flex-direction: column;
  flex: 1;
  height: 100%;
  min-height: 0;
  background: #1e1e1e;
  border-radius: 4px;
  overflow: hidden;
}

.pane.is-full {
  position: fixed;
  inset: 0;
  z-index: 1000;
  border-radius: 0;
}

.bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 8px;
  background: #2d2d2d;
  color: #ccc;
}

.tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow-x: auto;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 160px;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  white-space: nowrap;
}

.tab.active {
  background: #3c3c3c;
  color: #fff;
}

.tab-title {
  overflow: hidden;
  text-overflow: ellipsis;
}

.tab-close {
  opacity: 0.6;
}

.tab-close:hover {
  opacity: 1;
  color: #ff7875;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  border-radius: 4px;
  flex-shrink: 0;
}

.icon-btn:hover {
  background: #3c3c3c;
}

.stage {
  flex: 1;
  min-height: 0;
  height: 0;
  position: relative;
  background: #1e1e1e;
}

.term-host {
  position: absolute;
  inset: 0;
  padding: 4px;
}

.term-host.hidden {
  visibility: hidden;
  pointer-events: none;
}

.empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #888;
}
</style>
