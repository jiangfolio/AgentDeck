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

export type ProcessConn = {
  local: string
  remote: string
  status: string
  type: string
}

export type ProcessDetail = ProcessItem & {
  create_time: number
  num_threads: number
  num_fds: number
  memory_rss: number
  memory_vms: number
  memory_percent: number
  io_read_bytes: number
  io_write_bytes: number
  cwd: string
  exe: string
  cmdline_args: string[]
  open_files: string[]
  envs: string[]
  connections: ProcessConn[]
  partial_errors?: Record<string, string>
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

export type MarketApp = {
  id: string
  name: string
  summary: string
  category: string
  bin: string
  installed: boolean
}

export type MarketCatalog = {
  items: MarketApp[]
  total: number
  page: number
  page_size: number
  categories: string[]
}

export function marketCatalogApi(params: {
  tab?: string
  category?: string
  keyword?: string
  page: number
  page_size: number
  local_only?: boolean
}) {
  return apiGet<MarketCatalog>('/agent/market/catalog', { params })
}

export type MonitorSnapshot = {
  ts: number
  load: { supported: boolean; load1: number; load5: number; load15: number }
  cpu: { usage_percent: number; per_core: number[] }
  memory: { total_bytes: number; used_bytes: number; available_bytes: number; usage_percent: number }
  disk_io: {
    read_bytes_per_sec: number
    write_bytes_per_sec: number
    devices: { name: string; read_bytes_per_sec: number; write_bytes_per_sec: number }[]
  }
  net: {
    total_sent_bytes: number
    total_recv_bytes: number
    interfaces: {
      name: string
      bytes_sent_per_sec: number
      bytes_recv_per_sec: number
      bytes_sent: number
      bytes_recv: number
    }[]
  }
  top: { pid: number; user: string; name: string; cpu_percent: number }[]
}

export function monitorSnapshotApi(signal?: AbortSignal) {
  return apiGet<MonitorSnapshot>('/agent/monitor/snapshot', { signal })
}

export type FirewallRule = {
  id: string
  action: string
  protocol: string
  ip: string
  port: string
  description: string
  priority: number
  managed: boolean
}

export type FirewallStatus = {
  backend: string
  installed: boolean
  enabled: boolean
  version: string
  message: string
  os: string
  manager: string
  can_online: boolean
  can_offline: boolean
  offline_ext: string[]
  package: string
  is_root: boolean
}

export function firewallStatusApi() {
  return apiGet<FirewallStatus>('/agent/firewall/status')
}

export function firewallRulesApi(params: { keyword?: string; page: number; page_size: number }) {
  return apiGet<{ items: FirewallRule[]; total: number; page: number; page_size: number; message: string }>(
    '/agent/firewall/rules',
    { params },
  )
}

export function firewallSetEnabledApi(enabled: boolean) {
  return apiPost('/agent/firewall/set_enabled', { enabled })
}

export function firewallSaveApi(body: Partial<FirewallRule>) {
  return apiPost('/agent/firewall/save', body)
}

export function firewallRemoveApi(id: string) {
  return apiPost('/agent/firewall/remove', { id })
}

export function firewallResetApi() {
  return apiPost('/agent/firewall/reset', {})
}

export function firewallExportApi() {
  return apiGet<FirewallRule[]>('/agent/firewall/export')
}

export function firewallImportApi(mode: 'merge' | 'replace', rules: FirewallRule[]) {
  return apiPost('/agent/firewall/import', { mode, rules })
}

export function firewallInstallOnlineApi() {
  return apiPost('/agent/firewall/install_online', {}, { timeout: 360000 })
}

export function firewallInstallOfflineApi(file: File) {
  const body = new FormData()
  body.append('file', file)
  return apiPost('/agent/firewall/install_offline', body, { timeout: 360000 })
}

export type DiskPartition = {
  name: string
  size_bytes: number
  used_bytes: number
  free_bytes: number
  usage_percent: number
  mountpoint: string
  fstype: string
}

export type DiskItem = {
  name: string
  size_bytes: number
  system: boolean
  partitions: DiskPartition[]
}

export function diskListApi() {
  return apiGet<{ items: DiskItem[]; message: string }>('/agent/disk/list')
}

