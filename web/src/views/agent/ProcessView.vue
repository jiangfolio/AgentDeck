<script setup lang="ts">
import { computed, onMounted, reactive, shallowRef, ref, watch } from 'vue'
import { Modal, message } from 'antdv-next'
import {
  processDetailApi,
  processKillApi,
  processListApi,
  type ProcessDetail,
  type ProcessItem,
} from '@/api/agent'
import { usePolling } from '@/composables/usePolling'
import { useTableScrollY } from '@/composables/useTableScrollY'

const AUTO_KEY = 'agentdeck.process.auto_refresh'
const TREE_INDENT = 18

/** 树节点额外字段：画层级指引线用 */
type ProcessTreeNode = ProcessItem & {
  children?: ProcessTreeNode[]
  /** 深度，根为 0 */
  _depth: number
  /** 是否同级最后一个（决定 └ 还是 ├） */
  _isLast: boolean
  /** 祖先层是否继续竖线（同级后面还有兄弟） */
  _vline: boolean[]
  /** 是否有子节点（供展开按钮，避免依赖 children 引用抖动） */
  _hasChildren: boolean
}

const loading = ref(false)
/** 全量扁平列表；shallow 避免上千对象逐字段代理 */
const allItems = shallowRef<ProcessItem[]>([])
const total = ref(0)
/** 输入框状态（不直接触发建树） */
const query = reactive({
  keyword: '',
  user: '',
})
/** 已生效的筛选条件：点查询/重置后才建树 */
const applied = reactive({
  keyword: '',
  user: '',
})
/** 展开集合；默认全折叠，虚拟列表只渲染根节点 */
const expandedKeys = ref<(string | number)[]>([])
const expandedSet = computed(() => new Set(expandedKeys.value.map(Number)))

const autoRefresh = ref(localStorage.getItem(AUTO_KEY) === '1')
const detailOpen = ref(false)
const detailLoading = ref(false)
const detailTab = ref('basic')
const detail = ref<ProcessDetail | null>(null)
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 280, ellipsis: true },
  { title: 'PID', dataIndex: 'pid', key: 'pid', width: 90 },
  { title: 'PPID', dataIndex: 'ppid', key: 'ppid', width: 90 },
  { title: '用户', dataIndex: 'username', key: 'username', width: 100, ellipsis: true },
  { title: 'CPU%', dataIndex: 'cpu_percent', key: 'cpu_percent', width: 80 },
  { title: '内存%', dataIndex: 'mem_percent', key: 'mem_percent', width: 80 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90 },
  { title: '命令行', dataIndex: 'cmdline', key: 'cmdline', width: 240, ellipsis: true },
  { title: '操作', key: 'action', width: 180, fixed: 'right' as const },
]

/**
 * 按 PPID 组装森林，并标注指引线元数据。
 */
function buildProcessTree(list: ProcessItem[]): ProcessTreeNode[] {
  // 1. 复制节点并建 pid 索引
  const map = new Map<number, ProcessTreeNode>()
  for (const item of list) {
    map.set(item.pid, {
      ...item,
      children: undefined,
      _depth: 0,
      _isLast: true,
      _vline: [],
      _hasChildren: false,
    })
  }

  // 2. 按 ppid 挂子；父不在集合内则作为根
  const roots: ProcessTreeNode[] = []
  for (const node of map.values()) {
    const parent = map.get(node.ppid)
    if (parent && parent.pid !== node.pid) {
      if (!parent.children) parent.children = []
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  }

  // 3. 排序 + 标注 depth / 末子 / 祖先竖线
  const annotate = (nodes: ProcessTreeNode[], depth: number, ancestorVline: boolean[]) => {
    nodes.sort((a, b) => a.pid - b.pid)
    const last = nodes.length - 1
    for (let idx = 0; idx < nodes.length; idx++) {
      const n = nodes[idx]
      const isLast = idx === last
      n._depth = depth
      n._isLast = isLast
      n._vline = ancestorVline
      if (n.children?.length) {
        n._hasChildren = true
        annotate(n.children, depth + 1, [...ancestorVline, !isLast])
      } else {
        n._hasChildren = false
        delete n.children
      }
    }
  }
  annotate(roots, 0, [])
  return roots
}

/**
 * 筛选命中时保留祖先，避免树断裂。
 */
function filterWithAncestors(list: ProcessItem[], keyword: string, user: string): ProcessItem[] {
  // 1. 无条件直接返回
  const kw = keyword.trim().toLowerCase()
  const uf = user.trim().toLowerCase()
  if (!kw && !uf) return list

  // 2. pid 索引便于向上追溯
  const byPid = new Map<number, ProcessItem>()
  for (const item of list) byPid.set(item.pid, item)

  // 3. 命中节点 + 祖先
  const keep = new Set<number>()
  for (const item of list) {
    const nameHit =
      !kw ||
      item.name.toLowerCase().includes(kw) ||
      (item.cmdline || '').toLowerCase().includes(kw)
    const userHit = !uf || (item.username || '').toLowerCase() === uf
    if (!nameHit || !userHit) continue
    let cur: ProcessItem | undefined = item
    while (cur && !keep.has(cur.pid)) {
      keep.add(cur.pid)
      cur = byPid.get(cur.ppid)
    }
  }
  return list.filter((item) => keep.has(item.pid))
}

/** 树数据：仅随 allItems / applied 变化重建 */
const treeData = computed(() =>
  buildProcessTree(filterWithAncestors(allItems.value, applied.keyword, applied.user)),
)

const displayTotal = computed(() => {
  if (!applied.keyword.trim() && !applied.user.trim()) return total.value
  return filterWithAncestors(allItems.value, applied.keyword, applied.user).length
})

/**
 * 刷新后清理已消失的展开项；默认保持折叠（不自动展开根）。
 */
function pruneExpandedKeys(list: ProcessItem[]) {
  if (!expandedKeys.value.length) return
  const alive = new Set(list.map((i) => i.pid))
  const next = expandedKeys.value.filter((k) => alive.has(Number(k)))
  if (next.length !== expandedKeys.value.length) expandedKeys.value = next
}

/**
 * 拉取进程列表。silent 时不闪 loading，供自动刷新。
 */
async function load(signal?: AbortSignal, silent = false) {
  if (!silent) loading.value = true
  try {
    const data = await processListApi({ signal })
    allItems.value = data.items || []
    total.value = data.total || 0
    pruneExpandedKeys(allItems.value)
    if (!silent) recalc()
  } catch (e: any) {
    if (e?.code === 'ERR_CANCELED' || e?.name === 'CanceledError') return
    if (!silent) message.error(e?.message || '加载进程列表失败')
  } finally {
    if (!silent) loading.value = false
  }
}

const { start, stop } = usePolling(
  async (signal) => {
    await load(signal, true)
  },
  { interval: 5000, autoStart: false, immediate: true },
)

async function onManualRefresh() {
  await load(undefined, false)
}

function onSearch() {
  applied.keyword = query.keyword
  applied.user = query.user
  // 筛选后折叠，避免一次展开大量命中子树
  expandedKeys.value = []
  recalc()
}

function onReset() {
  query.keyword = ''
  query.user = ''
  applied.keyword = ''
  applied.user = ''
  expandedKeys.value = []
  recalc()
}

function statusColor(status: string) {
  const s = (status || '').toLowerCase()
  if (s === 'running' || s === 'r') return 'success'
  if (s === 'sleep' || s === 'sleeping' || s === 's' || s === 'idle' || s === 'i') return 'processing'
  if (s === 'zombie' || s === 'z' || s === 'dead') return 'error'
  if (s === 'stop' || s === 'stopped' || s === 't') return 'warning'
  return 'default'
}

/** 展开图标区样式：用渐变画竖线，减少每行 DOM 节点 */
function prefixStyle(record: ProcessTreeNode) {
  const depth = record._depth || 0
  const width = depth * TREE_INDENT + 18
  const images: string[] = []
  const positions: string[] = []
  const sizes: string[] = []
  const vline = record._vline || []
  for (let i = 0; i < vline.length; i++) {
    if (!vline[i]) continue
    images.push('linear-gradient(#d0d0d0, #d0d0d0)')
    positions.push(`${i * TREE_INDENT + 8}px 0`)
    sizes.push('1px 100%')
  }
  return {
    width: `${width}px`,
    backgroundImage: images.length ? images.join(',') : undefined,
    backgroundPosition: positions.length ? positions.join(',') : undefined,
    backgroundSize: sizes.length ? sizes.join(',') : undefined,
    backgroundRepeat: 'no-repeat',
  }
}

function onExpandToggle(record: ProcessTreeNode, e: MouseEvent) {
  e.stopPropagation()
  const pid = record.pid
  const set = expandedSet.value
  if (set.has(pid)) {
    expandedKeys.value = expandedKeys.value.filter((k) => Number(k) !== pid)
  } else {
    expandedKeys.value = [...expandedKeys.value, pid]
  }
}

watch(autoRefresh, (on) => {
  localStorage.setItem(AUTO_KEY, on ? '1' : '0')
  if (on) start()
  else stop()
})

async function openDetail(row: ProcessItem) {
  detailOpen.value = true
  detailTab.value = 'basic'
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await processDetailApi(row.pid)
  } catch (e: any) {
    message.error(e?.message || '获取详情失败')
    detailOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

function confirmKill(row: ProcessItem, tree: boolean) {
  Modal.confirm({
    title: tree ? `结束进程树 PID ${row.pid}？` : `结束进程 PID ${row.pid}？`,
    content: row.name,
    okType: 'danger',
    onOk: async () => {
      try {
        await processKillApi({ pid: row.pid, tree, force: true })
        message.success('已发送结束信号')
        await load()
      } catch (e: any) {
        message.error(e?.message || '结束失败')
      }
    },
  })
}

function formatBytes(n: number) {
  if (!n) return '0B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)}${u[i]}`
}

function formatTime(ms: number) {
  if (!ms) return '-'
  const d = new Date(ms)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

const basicRows = computed(() => {
  const d = detail.value
  if (!d) return []
  return [
    ['名称', d.name || '-', '状态', d.status || '-'],
    ['进程ID', String(d.pid), '父进程ID', String(d.ppid)],
    ['线程', String(d.num_threads || 0), '连接', String(d.connections?.length || 0)],
    ['磁盘读', formatBytes(d.io_read_bytes || 0), '磁盘写', formatBytes(d.io_write_bytes || 0)],
    ['用户', d.username || '-', '启动时间', formatTime(d.create_time)],
  ]
})

onMounted(() => {
  void load()
  if (autoRefresh.value) start()
})
</script>

<template>
  <div ref="wrapRef" class="ad-page-fill">
    <div class="ad-page-toolbar">
      <a-input
        v-model:value="query.keyword"
        allow-clear
        placeholder="请输入名称 / 命令行"
        style="width: 220px"
        @press-enter="onSearch"
      />
      <a-input
        v-model:value="query.user"
        allow-clear
        placeholder="请输入用户"
        style="width: 140px"
        @press-enter="onSearch"
      />
      <a-button type="primary" @click="onSearch">查询</a-button>
      <a-button @click="onReset">重置</a-button>
      <a-button :loading="loading" @click="onManualRefresh">刷新</a-button>
      <a-checkbox v-model:checked="autoRefresh">自动刷新</a-checkbox>
      <span class="total-hint">共 {{ displayTotal }} 条</span>
    </div>

    <div class="ad-page-table">
      <a-table
        row-key="pid"
        size="middle"
        virtual
        :loading="loading"
        :columns="columns"
        :data-source="treeData"
        :pagination="false"
        :scroll="{ y: scrollY, x: 1230 }"
        v-model:expanded-row-keys="expandedKeys"
        :indent-size="0"
        class="proc-tree-table"
      >
        <template #expandIcon="{ expanded, expandable, record }">
          <span class="proc-tree-prefix" :style="prefixStyle(record)">
            <i
              v-if="(record._depth || 0) > 0"
              class="proc-tree-elbow"
              :class="{ last: record._isLast }"
              :style="{ left: `${((record._depth || 1) - 1) * TREE_INDENT}px` }"
            />
            <button
              v-if="expandable || record._hasChildren"
              type="button"
              class="proc-tree-exp"
              :class="{ open: expanded }"
              :style="{ left: `${(record._depth || 0) * TREE_INDENT}px` }"
              @click="(e: MouseEvent) => onExpandToggle(record, e)"
            />
            <i
              v-else
              class="proc-tree-leaf"
              :style="{ left: `${(record._depth || 0) * TREE_INDENT + 5}px` }"
            />
          </span>
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'mem_percent'">
            {{ Number(record.mem_percent).toFixed(2) }}
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="statusColor(record.status)">{{ record.status || '-' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'cmdline'">
            <span class="cmdline-cell" :title="record.cmdline || ''">{{ record.cmdline || '-' }}</span>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a @click="openDetail(record)">详情</a>
              <a class="danger" @click="confirmKill(record, false)">结束</a>
              <a class="danger" @click="confirmKill(record, true)">结束树</a>
            </a-space>
          </template>
        </template>
      </a-table>
    </div>

    <a-modal
      v-model:open="detailOpen"
      title="进程详情"
      :footer="null"
      centered
      width="720px"
      :styles="{
        wrapper: { overflow: 'hidden' },
        body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' },
      }"
    >
      <a-spin :spinning="detailLoading">
        <template v-if="detail">
          <a-tabs v-model:active-key="detailTab">
            <a-tab-pane key="basic" tab="基本信息">
              <table class="desc-table">
                <tbody>
                  <tr v-for="(row, i) in basicRows" :key="i">
                    <th>{{ row[0] }}</th>
                    <td>{{ row[1] }}</td>
                    <th>{{ row[2] }}</th>
                    <td>{{ row[3] }}</td>
                  </tr>
                  <tr>
                    <th>启动命令</th>
                    <td colspan="3">{{ detail.cmdline || '-' }}</td>
                  </tr>
                </tbody>
              </table>
            </a-tab-pane>
            <a-tab-pane key="mem" tab="内存信息">
              <table class="desc-table">
                <tbody>
                  <tr>
                    <th>RSS</th>
                    <td>{{ formatBytes(detail.memory_rss) }}</td>
                    <th>VMS</th>
                    <td>{{ formatBytes(detail.memory_vms) }}</td>
                  </tr>
                  <tr>
                    <th>占用</th>
                    <td>{{ Number(detail.memory_percent || 0).toFixed(2) }}%</td>
                    <th>句柄</th>
                    <td>{{ detail.num_fds || 0 }}</td>
                  </tr>
                </tbody>
              </table>
            </a-tab-pane>
            <a-tab-pane key="files" tab="文件打开">
              <a-table
                size="small"
                :pagination="false"
                :data-source="(detail.open_files || []).map((p, i) => ({ key: i, path: p }))"
                :columns="[{ title: '路径', dataIndex: 'path', key: 'path', ellipsis: true }]"
                :locale="{ emptyText: detail.partial_errors?.open_files || '无数据或权限不足' }"
                :scroll="{ y: 280 }"
              />
            </a-tab-pane>
            <a-tab-pane key="env" tab="环境变量">
              <a-table
                size="small"
                :pagination="false"
                :data-source="(detail.envs || []).map((p, i) => ({ key: i, env: p }))"
                :columns="[{ title: '变量', dataIndex: 'env', key: 'env', ellipsis: true }]"
                :locale="{ emptyText: detail.partial_errors?.envs || '无数据或权限不足' }"
                :scroll="{ y: 280 }"
              />
            </a-tab-pane>
            <a-tab-pane key="net" tab="网络连接">
              <a-table
                size="small"
                :pagination="false"
                row-key="local"
                :data-source="detail.connections || []"
                :columns="[
                  { title: '类型', dataIndex: 'type', key: 'type', width: 80 },
                  { title: '本地', dataIndex: 'local', key: 'local', ellipsis: true },
                  { title: '远端', dataIndex: 'remote', key: 'remote', ellipsis: true },
                  { title: '状态', dataIndex: 'status', key: 'status', width: 110 },
                ]"
                :locale="{ emptyText: detail.partial_errors?.connections || '无数据或权限不足' }"
                :scroll="{ y: 280 }"
              />
            </a-tab-pane>
          </a-tabs>
        </template>
      </a-spin>
    </a-modal>
  </div>
</template>

<style scoped>
.total-hint {
  font-size: 13px;
  color: var(--ad-color-text-secondary);
}

.cmdline-cell {
  display: block;
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 树形指引线：竖线用 background 渐变，肘线少量 DOM */
.proc-tree-table :deep(.ant-table-tbody > tr > td:first-child) {
  position: relative;
  overflow: hidden;
}

.proc-tree-prefix {
  position: relative;
  display: inline-block;
  height: 14px;
  vertical-align: middle;
  flex-shrink: 0;
}

.proc-tree-elbow {
  position: absolute;
  top: -40px;
  bottom: -40px;
  width: 18px;
  pointer-events: none;
}

.proc-tree-elbow::before {
  content: '';
  position: absolute;
  left: 8px;
  top: 0;
  bottom: 50%;
  border-left: 1px solid #d0d0d0;
}

.proc-tree-elbow:not(.last)::before {
  bottom: 0;
}

.proc-tree-elbow::after {
  content: '';
  position: absolute;
  left: 8px;
  top: 50%;
  width: 10px;
  border-top: 1px solid #d0d0d0;
}

.proc-tree-exp {
  position: absolute;
  top: 50%;
  z-index: 1;
  width: 14px;
  height: 14px;
  margin: 0;
  padding: 0;
  border: 1px solid #bfbfbf;
  background: #fff;
  transform: translateY(-50%);
  cursor: pointer;
  line-height: 12px;
  font-size: 10px;
  color: #595959;
}

.proc-tree-exp::before {
  content: '+';
  display: block;
  text-align: center;
}

.proc-tree-exp.open::before {
  content: '−';
}

.proc-tree-leaf {
  position: absolute;
  top: 50%;
  z-index: 1;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #d0d0d0;
  transform: translateY(-50%);
  pointer-events: none;
}

.proc-tree-table :deep(.ant-table-row-indent) {
  padding: 0 !important;
}

.danger {
  color: #ff4d4f;
}

.desc-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.desc-table th,
.desc-table td {
  border: 1px solid var(--ad-color-border-secondary);
  padding: 8px 12px;
  text-align: left;
  word-break: break-all;
}

.desc-table th {
  width: 96px;
  background: #fafafa;
  color: var(--ad-color-text-secondary);
  font-weight: 500;
}
</style>
