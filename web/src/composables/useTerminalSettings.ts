import { reactive, watch } from 'vue'

const STORAGE_KEY = 'agentdeck.terminal.settings'

export type TerminalSettings = {
  fontSize: number
  theme: 'dark' | 'light'
  ignoreHostKey: boolean
}

const defaults: TerminalSettings = {
  fontSize: 14,
  theme: 'dark',
  ignoreHostKey: true,
}

function load(): TerminalSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...defaults }
    return { ...defaults, ...JSON.parse(raw) }
  } catch {
    return { ...defaults }
  }
}

export const terminalSettings = reactive<TerminalSettings>(load())

watch(
  terminalSettings,
  (v) => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(v))
  },
  { deep: true },
)

export const darkTheme = {
  background: '#1e1e1e',
  foreground: '#d4d4d4',
  cursor: '#d4d4d4',
  selectionBackground: '#264f78',
}

export const lightTheme = {
  background: '#ffffff',
  foreground: '#1f1f1f',
  cursor: '#1f1f1f',
  selectionBackground: '#add6ff',
}
