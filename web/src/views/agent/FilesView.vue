<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Modal, message } from 'antdv-next'
import {
  filesCompressApi,
  filesCopyApi,
  filesCreateApi,
  filesDownloadApi,
  filesExtractApi,
  filesListApi,
  filesMkdirApi,
  filesMoveApi,
  filesRemoveApi,
  filesRenameApi,
  filesRootsApi,
  filesTrashEmptyApi,
  filesTrashListApi,
  filesTrashPurgeApi,
  filesTrashRestoreApi,
  filesUploadApi,
  type FileItem,
  type TrashItem,
} from '@/api/agent'
import FileEditor from '@/components/FileEditor.vue'
import { useTableScrollY } from '@/composables/useTableScrollY'

const route = useRoute()
const loading = ref(false)
const roots = ref<string[]>([])
const currentPath = ref('')
const pathInput = ref('')
const parentPath = ref('')
const items = ref<FileItem[]>([])
const total = ref(0)
const selectedKeys = ref<string[]>([])
const query = reactive({ keyword: '', page: 1, page_size: 50 })
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef

const editorOpen = ref(false)
const editorPath = ref('')

const nameOpen = ref(false)
const nameMode = ref<'file' | 'dir' | 'rename'>('file')
const nameValue = ref('')
const renameTarget = ref<FileItem | null>(null)

const pasteOpen = ref(false)
const pasteMode = ref<'copy' | 'move'>('copy')
const pasteFrom = ref('')
const pasteTo = ref('')

const trashOpen = ref(false)
const trashLoading = ref(false)
const trashItems = ref<TrashItem[]>([])
const trashTotal = ref(0)
const trashSelected = ref<string[]>([])
const trashScrollY = ref(360)
const trashQuery = reactive({ keyword: '', page: 1, page_size: 20 })

const columns = [
  { title: '名称', key: 'name', ellipsis: true },
  { title: '权限', dataIndex: 'mode', key: 'mode', width: 90 },
  { title: '用户 / 用户组', dataIndex: 'owner', key: 'owner', width: 200, ellipsis: true },
  { title: '大小', key: 'size', width: 110 },
  { title: '修改时间', dataIndex: 'mod_time', key: 'mod_time', width: 170 },
  { title: '操作', key: 'action', width: 220, fixed: 'right' as const },
]

function formatSize(n: number, isDir: boolean) {
  if (isDir) return '-'
  if (!n) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 2)} ${u[i]}`
}

async function ensureRoots() {
  if (roots.value.length) return
  roots.value = (await filesRootsApi()) || []
}

async function load(path?: string) {
  loading.value = true
  try {
    await ensureRoots()
    const target = path ?? currentPath.value ?? roots.value[0] ?? ''
    const list = await filesListApi({
      path: target || undefined,
      keyword: query.keyword || undefined,
      page: query.page,
      page_size: query.page_size,
    })
    currentPath.value = list.path
    pathInput.value = list.path
    parentPath.value = list.parent || ''
    items.value = list.items || []
    total.value = list.total || 0
    selectedKeys.value = []
    recalc()
  } catch (e: any) {
    message.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  query.page = 1
  void load()
}

function onResetSearch() {
  query.keyword = ''
  query.page = 1
  void load()
}

function goParent() {
  if (!parentPath.value) return
  query.page = 1
  void load(parentPath.value)
}

/** 按输入框路径跳转，方便粘贴绝对路径直接访问。 */
function goPathInput() {
  const path = pathInput.value.trim()
  if (!path) {
    message.warning('请输入路径')
    return
  }
  query.page = 1
  void load(path)
}

function openItem(row: FileItem) {
  if (row.is_dir) {
    query.page = 1
    void load(row.path)
    return
  }
  editorPath.value = row.path
  editorOpen.value = true
}

async function onDownload(row: FileItem) {
  if (row.is_dir) {
    message.warning('不能下载目录')
    return
  }
  try {
    await filesDownloadApi(row.path, row.name)
  } catch (e: any) {
    message.error(e?.message || '下载失败')
  }
}

function openCreate(mode: 'file' | 'dir') {
  nameMode.value = mode
  nameValue.value = ''
  renameTarget.value = null
  nameOpen.value = true
}

function openRename(row: FileItem) {
  nameMode.value = 'rename'
  nameValue.value = row.name
  renameTarget.value = row
  nameOpen.value = true
}

async function submitName() {
  const name = nameValue.value.trim()
  if (!name) {
    message.warning('请输入名称')
    return
  }
  try {
    if (nameMode.value === 'dir') {
      await filesMkdirApi(currentPath.value, name)
    } else if (nameMode.value === 'file') {
      await filesCreateApi(currentPath.value, name)
    } else if (renameTarget.value) {
      await filesRenameApi(renameTarget.value.path, name)
    }
    message.success('已完成')
    nameOpen.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  }
}

function openPaste(mode: 'copy' | 'move', row: FileItem) {
  pasteMode.value = mode
  pasteFrom.value = row.path
  const sep = currentPath.value.includes('\\') ? '\\' : '/'
  const base = currentPath.value.endsWith('/') || currentPath.value.endsWith('\\')
    ? currentPath.value
    : currentPath.value + sep
  pasteTo.value = base + row.name + (mode === 'copy' ? '_copy' : '')
  pasteOpen.value = true
}

async function submitPaste() {
  try {
    if (pasteMode.value === 'copy') {
      await filesCopyApi(pasteFrom.value, pasteTo.value.trim())
    } else {
      await filesMoveApi(pasteFrom.value, pasteTo.value.trim())
    }
    message.success('已完成')
    pasteOpen.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  }
}

function confirmRemove(paths: string[], label: string) {
  Modal.confirm({
    title: `删除 ${label}？`,
    content: '将移入回收站，可稍后还原。',
    okType: 'danger',
    onOk: async () => {
      try {
        await filesRemoveApi(paths)
        message.success('已移入回收站')
        await load()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

async function loadTrash() {
  trashLoading.value = true
  try {
    const list = await filesTrashListApi({
      keyword: trashQuery.keyword || undefined,
      page: trashQuery.page,
      page_size: trashQuery.page_size,
    })
    trashItems.value = list.items || []
    trashTotal.value = list.total || 0
    trashSelected.value = []
  } catch (e: any) {
    message.error(e?.message || '加载回收站失败')
  } finally {
    trashLoading.value = false
  }
}

function openTrash() {
  trashQuery.keyword = ''
  trashQuery.page = 1
  // 预留给工具栏与分页，仅表格体滚动
  trashScrollY.value = Math.max(200, Math.min(420, window.innerHeight - 340))
  trashOpen.value = true
  void loadTrash()
}

function onTrashSearch() {
  trashQuery.page = 1
  void loadTrash()
}

function onTrashReset() {
  trashQuery.keyword = ''
  trashQuery.page = 1
  void loadTrash()
}

function onTrashSelect(keys: (string | number)[]) {
  trashSelected.value = keys.map(String)
}

async function restoreTrash(ids: string[]) {
  if (!ids.length) {
    message.warning('请先选择项目')
    return
  }
  try {
    await filesTrashRestoreApi(ids)
    message.success('已还原')
    await loadTrash()
    await load()
  } catch (e: any) {
    message.error(e?.message || '还原失败')
  }
}

function purgeTrash(ids: string[], label: string) {
  if (!ids.length) {
    message.warning('请先选择项目')
    return
  }
  Modal.confirm({
    title: `彻底删除 ${label}？`,
    content: '删除后不可恢复。',
    okType: 'danger',
    onOk: async () => {
      try {
        await filesTrashPurgeApi(ids)
        message.success('已彻底删除')
        await loadTrash()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

function emptyTrash() {
  Modal.confirm({
    title: '清空回收站？',
    content: '将彻底删除全部回收站内容，不可恢复。',
    okType: 'danger',
    onOk: async () => {
      try {
        await filesTrashEmptyApi()
        message.success('回收站已清空')
        await loadTrash()
      } catch (e: any) {
        message.error(e?.message || '清空失败')
      }
    },
  })
}

function onBatchRemove() {
  if (!selectedKeys.value.length) {
    message.warning('请先选择文件')
    return
  }
  confirmRemove(selectedKeys.value, `${selectedKeys.value.length} 项`)
}

function onUpload(file: File) {
  if (file.size > 64 * 1024 * 1024) {
    message.error('上传文件不能超过 64MB')
    return false
  }
  void (async () => {
    try {
      await filesUploadApi(currentPath.value, file)
      message.success('已上传')
      await load()
    } catch (e: any) {
      message.error(e?.message || '上传失败')
    }
  })()
  return false
}

async function copyPathText(path: string) {
  try {
    await navigator.clipboard.writeText(path)
    message.success('已复制路径')
  } catch {
    message.error('复制失败')
  }
}

function onPageChange(page: number, pageSize: number) {
  query.page = page
  query.page_size = pageSize
  void load()
}

function onSelectChange(keys: (string | number)[]) {
  selectedKeys.value = keys.map(String)
}

const createMenu = {
  items: [
    { key: 'file', label: '新建文件' },
    { key: 'dir', label: '新建目录' },
  ],
  onClick: ({ key }: { key: string | number }) => {
    openCreate(String(key) === 'dir' ? 'dir' : 'file')
  },
}

function isArchiveName(name: string) {
  const n = name.toLowerCase()
  return (
    n.endsWith('.zip') ||
    n.endsWith('.tar') ||
    n.endsWith('.tar.gz') ||
    n.endsWith('.tgz') ||
    n.endsWith('.tar.xz') ||
    n.endsWith('.txz')
  )
}

async function onExtract(row: FileItem) {
  try {
    const res = await filesExtractApi(row.path)
    message.success(`已解压到 ${res.dest}`)
    void load()
  } catch (e: any) {
    message.error(e?.message || '解压失败')
  }
}

async function onCompress(paths: string[], format: 'zip' | 'tar.gz') {
  if (!paths.length) {
    message.warning('请选择要压缩的项')
    return
  }
  try {
    const res = await filesCompressApi(paths, format)
    message.success(`已生成 ${res.path}`)
    void load()
  } catch (e: any) {
    message.error(e?.message || '压缩失败')
  }
}

function moreMenu(row: FileItem) {
  const items: { key?: string; label?: string; danger?: boolean; type?: 'divider' }[] = [
    { key: 'copy', label: '复制' },
    { key: 'move', label: '移动' },
    { key: 'rename', label: '重命名' },
    { key: 'path', label: '复制路径' },
  ]
  if (!row.is_dir && isArchiveName(row.name)) {
    items.push({ type: 'divider' }, { key: 'extract', label: '解压' })
  }
  items.push(
    { type: 'divider' },
    { key: 'zip', label: '压缩为 zip' },
    { key: 'tgz', label: '压缩为 tar.gz' },
    { type: 'divider' },
    { key: 'remove', label: '删除', danger: true },
  )
  return {
    items,
    onClick: ({ key }: { key: string | number }) => {
      const k = String(key)
      if (k === 'copy') openPaste('copy', row)
      else if (k === 'move') openPaste('move', row)
      else if (k === 'rename') openRename(row)
      else if (k === 'path') void copyPathText(row.path)
      else if (k === 'extract') void onExtract(row)
      else if (k === 'zip') void onCompress([row.path], 'zip')
      else if (k === 'tgz') void onCompress([row.path], 'tar.gz')
      else if (k === 'remove') confirmRemove([row.path], row.name)
    },
  }
}

/**
 * 支持 ?path= 打开指定目录（如应用市场安装完成后跳转维护）。
 */
function pathFromQuery() {
  const p = route.query.path
  return typeof p === 'string' ? p.trim() : ''
}

onMounted(() => {
  const initial = pathFromQuery()
  void load(initial || undefined)
})

watch(
  () => route.query.path,
  (p) => {
    if (route.name !== 'AgentFiles') return
    const path = typeof p === 'string' ? p.trim() : ''
    if (!path || path === currentPath.value) return
    query.page = 1
    void load(path)
  },
)
</script>

<template>
  <div ref="wrapRef" class="ad-page-fill">
    <div class="nav-bar">
      <a-space>
        <a-button :disabled="!parentPath" @click="goParent">上级</a-button>
        <a-button :loading="loading" @click="load()">刷新</a-button>
      </a-space>
      <a-input
        v-model:value="pathInput"
        class="path-input"
        allow-clear
        placeholder="输入或粘贴绝对路径后回车"
        @press-enter="goPathInput"
      />
      <a-button type="primary" @click="goPathInput">前往</a-button>
      <a-input
        v-model:value="query.keyword"
        allow-clear
        placeholder="搜索当前目录"
        style="width: 180px"
        @press-enter="onSearch"
      />
      <a-button @click="onSearch">搜索</a-button>
      <a-button @click="onResetSearch">重置</a-button>
    </div>

    <div class="ad-page-toolbar">
      <a-dropdown :trigger="['click']" :menu="createMenu">
        <a-button type="primary">新建</a-button>
      </a-dropdown>
      <a-upload :before-upload="onUpload" :show-upload-list="false">
        <a-button>上传</a-button>
      </a-upload>
      <a-button danger :disabled="!selectedKeys.length" @click="onBatchRemove">删除</a-button>
      <a-dropdown
        :trigger="['click']"
        :disabled="!selectedKeys.length"
        :menu="{
          items: [
            { key: 'zip', label: '压缩为 zip' },
            { key: 'tgz', label: '压缩为 tar.gz' },
          ],
          onClick: ({ key }: { key: string | number }) => {
            void onCompress(selectedKeys, String(key) === 'tgz' ? 'tar.gz' : 'zip')
          },
        }"
      >
        <a-button :disabled="!selectedKeys.length">压缩选中</a-button>
      </a-dropdown>
      <a-button @click="openTrash">回收站</a-button>
    </div>

    <div class="ad-page-table">
      <a-table
        row-key="path"
        size="middle"
        :loading="loading"
        :columns="columns"
        :data-source="items"
        :pagination="false"
        :scroll="{ y: scrollY, x: 1100 }"
        :row-selection="{
          selectedRowKeys: selectedKeys,
          onChange: onSelectChange,
        }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a class="name-link" :title="record.name" @click="openItem(record)">
              <span class="kind">{{ record.is_dir ? '目录' : '文件' }}</span>
              {{ record.name }}
            </a>
          </template>
          <template v-else-if="column.key === 'size'">
            {{ formatSize(record.size, record.is_dir) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a @click="openItem(record)">打开</a>
              <a v-if="!record.is_dir" @click="onDownload(record)">下载</a>
              <a
                v-if="!record.is_dir && isArchiveName(record.name)"
                @click="onExtract(record)"
              >
                解压
              </a>
              <a-dropdown :trigger="['click']" :menu="moreMenu(record)">
                <a @click.prevent>更多</a>
              </a-dropdown>
            </a-space>
          </template>
        </template>
      </a-table>
    </div>

    <div class="ad-page-pagination">
      <a-pagination
        v-model:current="query.page"
        v-model:page-size="query.page_size"
        :total="total"
        show-size-changer
        :show-total="(t: number) => `共 ${t} 条`"
        @change="onPageChange"
      />
    </div>

    <a-modal
      v-model:open="nameOpen"
      :title="nameMode === 'dir' ? '新建目录' : nameMode === 'file' ? '新建文件' : '重命名'"
      centered
      @ok="submitName"
    >
      <a-input v-model:value="nameValue" placeholder="请输入名称" maxlength="255" />
    </a-modal>

    <a-modal
      v-model:open="pasteOpen"
      :title="pasteMode === 'copy' ? '复制到' : '移动到'"
      centered
      @ok="submitPaste"
    >
      <p class="hint">源：{{ pasteFrom }}</p>
      <a-input v-model:value="pasteTo" placeholder="目标绝对路径" />
    </a-modal>

    <a-modal
      v-model:open="trashOpen"
      title="回收站"
      :footer="null"
      centered
      width="860px"
      :styles="{ body: { overflow: 'hidden' } }"
    >
      <div class="trash-body">
        <div class="trash-toolbar">
          <a-input
            v-model:value="trashQuery.keyword"
            allow-clear
            placeholder="搜索名称或原路径"
            style="width: 220px"
            @press-enter="onTrashSearch"
          />
          <a-button @click="onTrashSearch">搜索</a-button>
          <a-button @click="onTrashReset">重置</a-button>
          <a-button type="primary" :disabled="!trashSelected.length" @click="restoreTrash(trashSelected)">
            还原
          </a-button>
          <a-button
            danger
            :disabled="!trashSelected.length"
            @click="purgeTrash(trashSelected, `${trashSelected.length} 项`)"
          >
            彻底删除
          </a-button>
          <a-button danger :disabled="!trashTotal" @click="emptyTrash">清空</a-button>
        </div>
        <a-table
          row-key="id"
          size="small"
          :loading="trashLoading"
          :pagination="false"
          :data-source="trashItems"
          :scroll="{ y: trashScrollY, x: 780 }"
          :row-selection="{ selectedRowKeys: trashSelected, onChange: onTrashSelect }"
          :columns="[
            { title: '名称', dataIndex: 'name', key: 'name', width: 140, ellipsis: true },
            { title: '原路径', dataIndex: 'original_path', key: 'original_path', ellipsis: true },
            { title: '类型', key: 'kind', width: 70 },
            { title: '删除时间', dataIndex: 'deleted_at', key: 'deleted_at', width: 160 },
            { title: '操作', key: 'op', width: 150 },
          ]"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'kind'">{{ record.is_dir ? '目录' : '文件' }}</template>
            <template v-else-if="column.key === 'op'">
              <a-space>
                <a @click="restoreTrash([record.id])">还原</a>
                <a class="danger" @click="purgeTrash([record.id], record.name)">彻底删除</a>
              </a-space>
            </template>
          </template>
        </a-table>
        <div class="trash-page">
          <a-pagination
            v-model:current="trashQuery.page"
            v-model:page-size="trashQuery.page_size"
            :total="trashTotal"
            size="small"
            show-size-changer
            :show-total="(t: number) => `共 ${t} 条`"
            @change="loadTrash"
          />
        </div>
      </div>
    </a-modal>

    <FileEditor v-model:open="editorOpen" :path="editorPath" />
  </div>
</template>

<style scoped>
.nav-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.path-input {
  flex: 1;
  min-width: 160px;
}

.name-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  overflow: hidden;
}

.kind {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--ad-color-text-secondary);
  background: var(--ad-color-bg-layout);
  padding: 0 6px;
}

.hint {
  margin-bottom: 8px;
  color: var(--ad-color-text-secondary);
  word-break: break-all;
}

.trash-body {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.trash-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.trash-page {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
}

.danger {
  color: #ff4d4f;
}
</style>
