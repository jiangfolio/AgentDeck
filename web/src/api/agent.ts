import { apiGet, apiPost } from '@/api/http'

export type PageResult<T> = {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type ProcessItem = {
  pid: number
  ppid: number
  name: string
  username: string
  cpu_percent: number
  mem_percent: number
  status: string
  cmdline: string
}

export type ProcessDetail = ProcessItem & {
  create_time: number
  num_threads: number
  memory_rss: number
  memory_vms: number
  cwd: string
  exe: string
  cmdline_args: string[]
}

export type ProcessListQuery = {
  keyword?: string
  user?: string
  signal?: AbortSignal
}

export type ProcessListResult = {
  items: ProcessItem[]
  total: number
}

export function processListApi(q: ProcessListQuery = {}) {
  const { signal, ...params } = q
  return apiGet<ProcessListResult>('/agent/process/list', { params, signal })
}

export function processDetailApi(pid: number) {
  return apiGet<ProcessDetail>('/agent/process/detail', { params: { pid } })
}

export function processKillApi(body: {
  pid: number
  tree?: boolean
  force?: boolean
  confirm_self?: boolean
}) {
  return apiPost('/agent/process/kill', body)
}

export type CronJob = {
  id: string
  name: string
  expr: string
  command: string
  work_dir: string
  enabled: boolean
  created_at: string
  updated_at: string
  next_run_at: number
}

export type CronRun = {
  ts: string
  exit_code: number
  duration_ms: number
  stdout: string
  stderr: string
}

export function cronListApi(params: { page: number; page_size: number; enabled?: boolean }) {
  return apiGet<PageResult<CronJob>>('/agent/cron/list', { params })
}

export function cronSaveApi(body: {
  id?: string
  name: string
  expr: string
  command: string
  work_dir?: string
  enabled?: boolean
}) {
  return apiPost<CronJob>('/agent/cron/save', body)
}

export function cronRemoveApi(id: string) {
  return apiPost('/agent/cron/remove', { id })
}

export function cronSetEnabledApi(id: string, enabled: boolean) {
  return apiPost('/agent/cron/set_enabled', { id, enabled })
}

export function cronRunsApi(id: string) {
  return apiGet<CronRun[]>('/agent/cron/runs', { params: { id } })
}

export type StartupItem = {
  id: string
  name: string
  command: string
  work_dir: string
  enabled: boolean
  managed: boolean
  source: string
  read_only: boolean
  kind: string
}

export type StartupSelf = {
  enabled: boolean
  exe_path: string
}

export function startupSelfApi() {
  return apiGet<StartupSelf>('/agent/startup/self')
}

export function startupSelfSetApi(enabled: boolean) {
  return apiPost('/agent/startup/self_set', { enabled })
}

export function startupListApi(params: { page: number; page_size: number }) {
  return apiGet<PageResult<StartupItem>>('/agent/startup/list', { params })
}

export function startupSaveApi(body: {
  id?: string
  name: string
  command: string
  work_dir?: string
  enabled?: boolean
}) {
  return apiPost('/agent/startup/save', body)
}

export function startupRemoveApi(id: string) {
  return apiPost('/agent/startup/remove', { id })
}

export function startupSetEnabledApi(id: string, enabled: boolean) {
  return apiPost('/agent/startup/set_enabled', { id, enabled })
}

export type CommandRunResult = {
  exit_code: number
  stdout: string
  stderr: string
  duration_ms: number
}

/** 试运行命令（如 echo Hello） */
export function commandRunApi(body: { command: string; work_dir?: string }) {
  return apiPost<CommandRunResult>('/agent/command/run', body)
}

/** 跨平台 Hello 示例命令 */
export const HELLO_COMMAND = 'echo "Hello AgentDeck"'

