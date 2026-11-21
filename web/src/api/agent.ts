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

export type FileItem = {
  name: string
  path: string
  is_dir: boolean
  size: number
  mode: string
  mod_time: string
  owner: string
}

export type FileListResult = {
  path: string
  parent: string
  items: FileItem[]
  total: number
  page: number
  page_size: number
}

export type FileReadResult = {
  path: string
  name: string
  content: string
  size: number
  language: string
}

export function filesRootsApi() {
  return apiGet<string[]>('/agent/files/roots')
}

export function filesListApi(params: { path?: string; keyword?: string; page: number; page_size: number }) {
  return apiGet<FileListResult>('/agent/files/list', { params })
}

export function filesReadApi(path: string) {
  return apiGet<FileReadResult>('/agent/files/read', { params: { path } })
}

export function filesWriteApi(path: string, content: string) {
  return apiPost('/agent/files/write', { path, content }, { timeout: 120000 })
}

export function filesMkdirApi(path: string, name: string) {
  return apiPost('/agent/files/mkdir', { path, name })
}

export function filesCreateApi(path: string, name: string) {
  return apiPost('/agent/files/create', { path, name })
}

export function filesRenameApi(path: string, new_name: string) {
  return apiPost('/agent/files/rename', { path, new_name })
}

export function filesRemoveApi(paths: string[]) {
  return apiPost('/agent/files/remove', { paths })
}

export type TrashItem = {
  id: string
  name: string
  original_path: string
  is_dir: boolean
  size: number
  deleted_at: string
}

export function filesTrashListApi(params: { keyword?: string; page: number; page_size: number }) {
  return apiGet<{ items: TrashItem[]; total: number; page: number; page_size: number }>(
    '/agent/files/trash/list',
    { params },
  )
}

export function filesTrashRestoreApi(ids: string[]) {
  return apiPost('/agent/files/trash/restore', { ids })
}

export function filesTrashPurgeApi(ids: string[]) {
  return apiPost('/agent/files/trash/purge', { ids })
}

export function filesTrashEmptyApi() {
  return apiPost('/agent/files/trash/empty', {})
}

export function filesCopyApi(from: string, to: string) {
  return apiPost('/agent/files/copy', { from, to })
}

export function filesMoveApi(from: string, to: string) {
  return apiPost('/agent/files/move', { from, to })
}

export function filesUploadApi(path: string, file: File) {
  const body = new FormData()
  body.append('path', path)
  body.append('file', file)
  return apiPost('/agent/files/upload', body, { timeout: 360000 })
}

// —— 终端主机 / 快速命令 / 脚本库 ——

export type TermHostGroup = {
  id: string
  name: string
  sort: number
  created_at: string
}

export type TermHost = {
  id: string
  group_id: string
  group_name: string
  title: string
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  has_secret: boolean
  description: string
  sort: number
  created_at: string
  updated_at: string
}

export function termHostGroupsApi() {
  return apiGet<TermHostGroup[]>('/agent/term/host/groups')
}

export function termHostGroupSaveApi(body: { id?: string; name: string; sort?: number }) {
  return apiPost<TermHostGroup>('/agent/term/host/group/save', body)
}

export function termHostGroupRemoveApi(id: string) {
  return apiPost('/agent/term/host/group/remove', { id })
}

export function termHostListApi(params: {
  page: number
  page_size: number
  keyword?: string
  group_id?: string
}) {
  return apiGet<PageResult<TermHost>>('/agent/term/host/list', { params })
}

export function termHostSaveApi(body: {
  id?: string
  group_id?: string
  title: string
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  secret?: string
  passphrase?: string
  description?: string
  sort?: number
}) {
  return apiPost<TermHost>('/agent/term/host/save', body)
}

export function termHostRemoveApi(ids: string[]) {
  return apiPost('/agent/term/host/remove', { ids })
}

export type TermHostTestResult = {
  ok: boolean
  latency_ms: number
  message: string
}

export function termHostTestApi(body: {
  id?: string
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  secret?: string
  passphrase?: string
}) {
  return apiPost<TermHostTestResult>('/agent/term/host/test', body)
}

export type TermQuickCmd = {
  id: string
  group_name: string
  name: string
  command: string
  sort: number
  created_at: string
  updated_at: string
}

export function termCmdListApi(params: {
  page: number
  page_size: number
  keyword?: string
  group_name?: string
}) {
  return apiGet<PageResult<TermQuickCmd>>('/agent/term/cmd/list', { params })
}

export function termCmdGroupsApi() {
  return apiGet<string[]>('/agent/term/cmd/groups')
}

export function termCmdSaveApi(body: {
  id?: string
  group_name?: string
  name: string
  command: string
  sort?: number
}) {
  return apiPost<TermQuickCmd>('/agent/term/cmd/save', body)
}

export function termCmdRemoveApi(ids: string[]) {
  return apiPost('/agent/term/cmd/remove', { ids })
}

export type ScriptGroup = {
  id: string
  name: string
  sort: number
  created_at: string
}

export type ScriptItem = {
  id: string
  group_id: string
  group_name: string
  name: string
  content: string
  interactive: boolean
  description: string
  created_at: string
  updated_at: string
}

export type ScriptRunResult = {
  interactive: boolean
  message?: string
  exit_code?: number
  stdout?: string
  stderr?: string
  duration_ms?: number
  content?: string
}

export function scriptGroupsApi() {
  return apiGet<ScriptGroup[]>('/agent/script/groups')
}

export function scriptGroupSaveApi(body: { id?: string; name: string; sort?: number }) {
  return apiPost<ScriptGroup>('/agent/script/group/save', body)
}

export function scriptGroupRemoveApi(id: string) {
  return apiPost('/agent/script/group/remove', { id })
}

export function scriptListApi(params: {
  page: number
  page_size: number
  keyword?: string
  group_id?: string
}) {
  return apiGet<PageResult<ScriptItem>>('/agent/script/list', { params })
}

export function scriptSaveApi(body: {
  id?: string
  group_id?: string
  name: string
  content: string
  interactive?: boolean
  description?: string
}) {
  return apiPost<ScriptItem>('/agent/script/save', body)
}

export function scriptRemoveApi(ids: string[]) {
  return apiPost('/agent/script/remove', { ids })
}

export function scriptRunApi(body: { id: string; host_id?: string }) {
  return apiPost<ScriptRunResult>('/agent/script/run', body)
}

/** 构建终端 WebSocket URL（同域 cookie 鉴权）。 */
export function terminalWsUrl(params: {
  kind: 'local' | 'ssh'
  host_id?: string
  cols: number
  rows: number
  ignore_host?: boolean
}) {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const q = new URLSearchParams({
    kind: params.kind,
    cols: String(params.cols),
    rows: String(params.rows),
    ignore_host: params.ignore_host === false ? '0' : '1',
  })
  if (params.host_id) q.set('host_id', params.host_id)
  return `${proto}//${location.host}/api/agent/terminal/ws?${q.toString()}`
}

/** 触发浏览器下载；走 cookie 会话。 */
export async function filesDownloadApi(path: string, filename: string) {
  const res = await fetch(`/api/agent/files/download?path=${encodeURIComponent(path)}`, {
    credentials: 'include',
  })
  const ct = res.headers.get('content-type') || ''
  if (!res.ok || ct.includes('application/json')) {
    let msg = '下载失败'
    try {
      const body = await res.json()
      if (body?.message) msg = body.message
    } catch {
      /* ignore */
    }
    throw new Error(msg)
  }
  const blob = await res.blob()
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = filename || 'download'
  a.click()
  URL.revokeObjectURL(a.href)
}

