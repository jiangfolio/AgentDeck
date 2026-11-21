import { ref } from 'vue'

export type OpenSessionRequest = {
  kind: 'local' | 'ssh'
  hostId?: string
  title: string
  /** 交互脚本等内容，连接成功后写入 */
  inject?: string
}

/** 跨 Tab 请求打开终端会话（主机「连接」、脚本交互执行）。 */
const pending = ref<OpenSessionRequest | null>(null)

export function requestOpenSession(req: OpenSessionRequest) {
  pending.value = { ...req }
}

export function takePendingSession() {
  const v = pending.value
  pending.value = null
  return v
}

export function useTerminalBridge() {
  return { pending, requestOpenSession, takePendingSession }
}
