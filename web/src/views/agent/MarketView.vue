<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'antdv-next'
import {
  marketCatalogApi,
  marketCatalogRemoveApi,
  marketCatalogSaveApi,
  marketClearDirApi,
  marketOpenDirApi,
  marketStartApi,
  marketStopApi,
  marketUploadApi,
  type MarketApp,
} from '@/api/agent'
import { useTableScrollY } from '@/composables/useTableScrollY'

const router = useRouter()
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef

const loading = ref(false)
const items = ref<MarketApp[]>([])
const total = ref(0)
const categories = ref<string[]>([])
const query = reactive({
  tab: 'all',
  category: '全部',
  keyword: '',
  page: 1,
  page_size: 20,
})

const uploadApp = ref<MarketApp | null>(null)
const uploadInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)

const formOpen = ref(false)
const formEditing = ref(false)
const saving = ref(false)
const form = reactive({
  id: '',
  name: '',
  summary: '',
  category: '',
  bin: '',
  startText: '{bin}',
  runnable: true,
  sort: 0,
})

const columns = [
  { title: '名称', key: 'name', width: 160, ellipsis: true },
  { title: 'ID', dataIndex: 'id', key: 'id', width: 120, ellipsis: true },
  { title: '分类', key: 'category', width: 110 },
  { title: '简介', dataIndex: 'summary', key: 'summary', ellipsis: true },
  { title: '目录', dataIndex: 'app_dir', key: 'app_dir', ellipsis: true },
  { title: '状态', key: 'status', width: 90 },
  { title: '操作', key: 'op', width: 320, fixed: 'right' as const },
]

const tagColors = ['blue', 'green', 'orange', 'purple', 'cyan', 'geekblue', 'gold']

function tagColor(name: string) {
  let n = 0
  for (const ch of name) n += ch.charCodeAt(0)
  return tagColors[n % tagColors.length]
}

/**
 * 拉取应用目录。
 */
async function load() {
  loading.value = true
  try {
    const data = await marketCatalogApi({
      tab: query.tab,
      category: query.category === '全部' ? '' : query.category,
      keyword: query.keyword || undefined,
      page: query.page,
      page_size: query.page_size,
    })
    items.value = data.items || []
    total.value = data.total || 0
    categories.value = data.categories || []
    recalc()
  } catch (e: any) {
    message.error(e?.message || '加载应用目录失败')
  } finally {
    loading.value = false
  }
}

function onTab(key: string) {
  query.tab = key
  query.page = 1
  void load()
}

function onSearch() {
  query.page = 1
  void load()
}

function onCategory(name: string) {
  query.category = name
  query.page = 1
  void load()
}

function openCreate() {
  formEditing.value = false
  form.id = ''
  form.name = ''
  form.summary = ''
  form.category = ''
  form.bin = ''
  form.startText = '{bin}'
  form.runnable = true
  form.sort = 0
  formOpen.value = true
}

function openEdit(app: MarketApp) {
  formEditing.value = true
  form.id = app.id
  form.name = app.name
  form.summary = app.summary || ''
  form.category = app.category || ''
  form.bin = app.bin || ''
  form.startText = (app.start || []).join('\n')
  form.runnable = !!app.runnable
  form.sort = app.sort || 0
  formOpen.value = true
}

async function submitCatalog() {
  if (!form.id.trim() || !form.name.trim() || !form.bin.trim()) {
    message.warning('请填写 id、名称与二进制路径')
    return Promise.reject()
  }
  saving.value = true
  try {
    const start = form.runnable
      ? form.startText
          .split('\n')
          .map((s) => s.trim())
          .filter(Boolean)
      : []
    await marketCatalogSaveApi({
      id: form.id.trim(),
      name: form.name.trim(),
      summary: form.summary.trim(),
      category: form.category.trim(),
      bin: form.bin.trim(),
      start,
      runnable: form.runnable,
      sort: form.sort || 0,
    })
    message.success(formEditing.value ? '已保存' : '已创建')
    formOpen.value = false
    void load()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
    return Promise.reject(e)
  } finally {
    saving.value = false
  }
}

function onDeleteCatalog(app: MarketApp) {
  Modal.confirm({
    title: `删除应用 ${app.name}？`,
    content: '将删除目录定义与应用目录下的全部文件，不可恢复。',
    okText: '删除',
    okType: 'danger',
    async onOk() {
      try {
        await marketCatalogRemoveApi(app.id)
        message.success('已删除')
        void load()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

async function openAppDir(app: MarketApp) {
  try {
    const data = await marketOpenDirApi(app.id)
    await router.push({ name: 'AgentFiles', query: { path: data.app_dir } })
  } catch (e: any) {
    message.error(e?.message || '打开目录失败')
  }
}

function pickUpload(app: MarketApp) {
  uploadApp.value = app
  uploadInput.value?.click()
}

async function onUploadChange(ev: Event) {
  const input = ev.target as HTMLInputElement
  const files = input.files ? Array.from(input.files) : []
  input.value = ''
  const app = uploadApp.value
  if (!app || !files.length) return
  uploading.value = true
  try {
    const res = await marketUploadApi(app.id, files)
    message.success(`已上传 ${res.files?.length || files.length} 个文件，正在打开目录`)
    await router.push({ name: 'AgentFiles', query: { path: res.app_dir } })
  } catch (e: any) {
    message.error(e?.message || '上传失败')
  } finally {
    uploading.value = false
    uploadApp.value = null
  }
}

function onClearDir(app: MarketApp) {
  Modal.confirm({
    title: `清空 ${app.name} 目录？`,
    content: '将停止进程并删除应用目录内全部文件，目录本身保留。',
    okText: '清空',
    okType: 'danger',
    async onOk() {
      try {
        await marketClearDirApi(app.id)
        message.success('已清空')
        void load()
      } catch (e: any) {
        message.error(e?.message || '清空失败')
      }
    },
  })
}

async function onStart(app: MarketApp) {
  try {
    await marketStartApi(app.id)
    message.success('已启动')
    void load()
  } catch (e: any) {
    message.error(e?.message || '启动失败')
  }
}

async function onStop(app: MarketApp) {
  try {
    await marketStopApi(app.id)
    message.success('已停止')
    void load()
  } catch (e: any) {
    message.error(e?.message || '停止失败')
  }
}

onMounted(load)
</script>

<template>
  <div ref="wrapRef" class="ad-page-fill market-page">
    <input
      ref="uploadInput"
      type="file"
      multiple
      class="hidden-input"
      @change="onUploadChange"
    />

    <div class="market-head">
      <a-tabs :active-key="query.tab" @change="(k: string) => onTab(k)">
        <a-tab-pane key="all" tab="全部" />
        <a-tab-pane key="settings" tab="目录管理" />
      </a-tabs>

      <div class="cats">
        <a-button
          size="small"
          :type="query.category === '全部' ? 'primary' : 'default'"
          @click="onCategory('全部')"
        >
          全部
        </a-button>
        <a-button
          v-for="c in categories"
          :key="c"
          size="small"
          :type="query.category === c ? 'primary' : 'default'"
          @click="onCategory(c)"
        >
          {{ c }}
        </a-button>
      </div>

      <div class="ad-page-toolbar">
        <a-button type="primary" @click="openCreate">新建应用</a-button>
        <span class="hint">应用目录在 ~/download 下，临时目录为 ~/download/temp</span>
        <a-input
          v-model:value="query.keyword"
          allow-clear
          placeholder="搜索"
          style="width: 220px; margin-left: auto"
          @press-enter="onSearch"
        />
        <a-button type="primary" @click="onSearch">搜索</a-button>
      </div>
    </div>

    <div class="ad-page-table">
      <a-table
        row-key="id"
        size="small"
        :loading="loading || uploading"
        :columns="columns"
        :data-source="items"
        :pagination="false"
        :scroll="{ x: 1100, y: scrollY }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <span class="name-cell">{{ record.name }}</span>
          </template>
          <template v-else-if="column.key === 'category'">
            <a-tag v-if="record.category" :color="tagColor(record.category)">{{ record.category }}</a-tag>
            <span v-else>-</span>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag v-if="record.running" color="processing">运行中</a-tag>
            <span v-else-if="record.runnable" class="muted">已停止</span>
            <span v-else class="muted">仅目录</span>
          </template>
          <template v-else-if="column.key === 'op'">
            <a-space wrap>
              <template v-if="query.tab === 'settings'">
                <a @click="openEdit(record)">编辑</a>
                <a class="danger" @click="onDeleteCatalog(record)">删除</a>
              </template>
              <template v-else>
                <a @click="pickUpload(record)">上传文件</a>
                <a @click="openAppDir(record)">打开目录</a>
                <a v-if="record.runnable && !record.running" @click="onStart(record)">启动</a>
                <a v-if="record.runnable && record.running" @click="onStop(record)">停止</a>
                <a @click="openEdit(record)">编辑</a>
                <a class="danger" @click="onClearDir(record)">清空</a>
              </template>
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
        :page-size-options="['20', '40', '60', '100']"
        show-size-changer
        :show-total="(t: number) => `共 ${t} 条`"
        @change="load"
      />
    </div>

    <a-modal
      v-model:open="formOpen"
      :title="formEditing ? '编辑应用' : '新建应用'"
      width="560px"
      :confirm-loading="saving"
      ok-text="保存"
      centered
      :styles="{
        wrapper: { overflow: 'hidden' },
        body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' },
      }"
      destroy-on-hidden
      @ok="submitCatalog"
    >
      <div class="install-form">
        <div class="field">
          <div class="label">应用 ID</div>
          <a-input v-model:value="form.id" :disabled="formEditing" placeholder="小写，如 redis" />
        </div>
        <div class="field">
          <div class="label">名称</div>
          <a-input v-model:value="form.name" placeholder="显示名称" />
        </div>
        <div class="field">
          <div class="label">分类</div>
          <a-input v-model:value="form.category" placeholder="如 数据库" />
        </div>
        <div class="field">
          <div class="label">简介</div>
          <a-textarea v-model:value="form.summary" :rows="2" />
        </div>
        <div class="field">
          <div class="label">二进制相对路径（相对应用目录）</div>
          <a-input v-model:value="form.bin" placeholder="bin/redis-server" />
        </div>
        <div class="field">
          <div class="label">可启停</div>
          <a-switch v-model:checked="form.runnable" />
        </div>
        <div v-if="form.runnable" class="field">
          <div class="label">启动参数（每行一项，可用 {bin} {install_dir}）</div>
          <a-textarea v-model:value="form.startText" :rows="3" placeholder="{bin}" />
        </div>
        <div class="field">
          <div class="label">排序</div>
          <a-input-number v-model:value="form.sort" style="width: 100%" />
        </div>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.market-page {
  background: var(--ad-color-bg-container);
}

.hidden-input {
  display: none;
}

.market-head {
  flex-shrink: 0;
}

.market-head :deep(.ant-tabs-content-holder) {
  display: none;
}

.cats {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.hint {
  color: var(--ad-color-text-secondary);
  font-size: 13px;
  margin-left: 12px;
}

.name-cell {
  font-weight: 600;
}

.muted {
  color: var(--ad-color-text-secondary);
  font-size: 13px;
}

.danger {
  color: #ff4d4f;
}

.install-form .field {
  margin-bottom: 16px;
}

.install-form .label {
  margin-bottom: 6px;
  font-weight: 500;
}
</style>
