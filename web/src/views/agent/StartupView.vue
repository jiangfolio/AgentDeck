<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Modal, message } from 'antdv-next'
import {
  HELLO_COMMAND,
  commandRunApi,
  startupListApi,
  startupRemoveApi,
  startupSaveApi,
  startupSelfApi,
  startupSelfSetApi,
  startupSetEnabledApi,
  type CommandRunResult,
  type StartupItem,
  type StartupSelf,
} from '@/api/agent'
import CommandEditor from '@/components/CommandEditor.vue'
import { useTableScrollY } from '@/composables/useTableScrollY'

const loading = ref(false)
const items = ref<StartupItem[]>([])
const total = ref(0)
const query = reactive({ page: 1, page_size: 20 })
const self = ref<StartupSelf | null>(null)

const formOpen = ref(false)
const saving = ref(false)
const testing = ref(false)
const testResult = ref<CommandRunResult | null>(null)
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef
const form = reactive({
  id: '',
  name: '',
  command: '',
  work_dir: '',
  enabled: true,
})

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 180 },
  { title: '类型', key: 'kind', width: 100 },
  { title: '命令', dataIndex: 'command', key: 'command', ellipsis: true },
  { title: '来源', dataIndex: 'source', key: 'source', width: 140 },
  { title: '启用', key: 'enabled', width: 90 },
  { title: '操作', key: 'action', width: 160 },
]

const kindLabel: Record<string, string> = {
  self: '自身',
  custom: '自定义',
  platform: '系统',
}

const kindColor: Record<string, string> = {
  self: 'blue',
  custom: 'green',
  platform: 'orange',
}

function sourceColor(source: string) {
  const s = (source || '').toLowerCase()
  if (s === 'self') return 'blue'
  if (s === 'custom') return 'green'
  if (s.includes('xdg')) return 'cyan'
  if (s.includes('launch')) return 'purple'
  if (s.includes('schtask')) return 'geekblue'
  return 'default'
}

async function load() {
  loading.value = true
  try {
    const [list, selfStatus] = await Promise.all([
      startupListApi({ page: query.page, page_size: query.page_size }),
      startupSelfApi(),
    ])
    items.value = list.items || []
    total.value = list.total || 0
    self.value = selfStatus
    recalc()
  } catch (e: any) {
    message.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function onToggleSelf(enabled: boolean) {
  try {
    await startupSelfSetApi(enabled)
    message.success(enabled ? '已开启开机自启' : '已关闭开机自启')
    await load()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  }
}

function openCreate() {
  Object.assign(form, {
    id: '',
    name: '',
    command: '',
    work_dir: '',
    enabled: true,
  })
  testResult.value = null
  formOpen.value = true
}

function openEdit(row: StartupItem) {
  Object.assign(form, {
    id: row.id,
    name: row.name,
    command: row.command,
    work_dir: row.work_dir || '',
    enabled: row.enabled,
  })
  testResult.value = null
  formOpen.value = true
}

function fillHello() {
  form.command = HELLO_COMMAND
  if (!form.name.trim()) form.name = 'Hello 测试'
  testResult.value = null
}

async function onTestRun() {
  if (!form.command.trim()) {
    message.warning('请先填写命令')
    return
  }
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await commandRunApi({
      command: form.command,
      work_dir: form.work_dir || undefined,
    })
  } catch (e: any) {
    message.error(e?.message || '试运行失败')
  } finally {
    testing.value = false
  }
}

async function onSave() {
  if (!form.name.trim() || !form.command.trim()) {
    message.warning('请填写名称与命令')
    return
  }
  saving.value = true
  try {
    await startupSaveApi({
      id: form.id || undefined,
      name: form.name.trim(),
      command: form.command,
      work_dir: form.work_dir,
      enabled: form.enabled,
    })
    message.success('已保存')
    formOpen.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(row: StartupItem, enabled: boolean) {
  if (row.kind === 'self') {
    await onToggleSelf(enabled)
    return
  }
  if (row.read_only) {
    message.warning('只读项不可修改')
    return
  }
  try {
    await startupSetEnabledApi(row.id, enabled)
    message.success(enabled ? '已启用' : '已禁用')
    await load()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  }
}

function confirmRemove(row: StartupItem) {
  Modal.confirm({
    title: `删除自启条目「${row.name}」？`,
    okType: 'danger',
    onOk: async () => {
      try {
        await startupRemoveApi(row.id)
        message.success('已删除')
        await load()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

function onPageChange(page: number, pageSize: number) {
  query.page = page
  query.page_size = pageSize
  void load()
}

onMounted(load)
</script>

<template>
  <div ref="wrapRef" class="ad-page-fill">
    <div class="self-card">
      <div>
        <div class="self-title">AgentDeck 开机自启</div>
        <div class="self-path">{{ self?.exe_path || '-' }}</div>
      </div>
      <a-switch
        :checked="!!self?.enabled"
        checked-children="开"
        un-checked-children="关"
        @change="(v: boolean) => onToggleSelf(v)"
      />
    </div>

    <div class="ad-page-toolbar">
      <a-button type="primary" @click="openCreate">新建自启条目</a-button>
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>

    <div class="ad-page-table">
      <a-table
        row-key="id"
        size="middle"
        :loading="loading"
        :columns="columns"
        :data-source="items"
        :pagination="false"
        :scroll="{ y: scrollY, x: 'max-content' }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'kind'">
            <a-tag :color="kindColor[record.kind] || 'default'">
              {{ kindLabel[record.kind] || record.kind }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'source'">
            <a-tag :color="sourceColor(record.source)">{{ record.source }}</a-tag>
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-switch
              :checked="record.enabled"
              :disabled="record.read_only && record.kind !== 'self'"
              @change="(v: boolean) => toggleEnabled(record, v)"
            />
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space v-if="record.kind === 'custom'">
              <a @click="openEdit(record)">编辑</a>
              <a class="danger" @click="confirmRemove(record)">删除</a>
            </a-space>
            <span v-else class="muted">—</span>
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
      v-model:open="formOpen"
      :title="form.id ? '编辑自启条目' : '新建自启条目'"
      :confirm-loading="saving"
      centered
      width="720px"
      :styles="{
        wrapper: { overflow: 'hidden' },
        body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' },
      }"
      destroy-on-hidden
      @ok="onSave"
    >
      <a-form layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="form.name" maxlength="128" placeholder="请输入条目名称" />
        </a-form-item>
        <a-form-item label="命令" required>
          <div class="cmd-actions">
            <a-button size="small" @click="fillHello">Hello 示例</a-button>
            <a-button size="small" type="primary" ghost :loading="testing" @click="onTestRun">
              试运行
            </a-button>
          </div>
          <CommandEditor v-model="form.command" language="shell" height="180px" />
        </a-form-item>
        <a-form-item v-if="testResult" label="试运行结果">
          <div class="run-box">
            <div class="run-meta">
              <a-tag :color="testResult.exit_code === 0 ? 'success' : 'error'">
                exit {{ testResult.exit_code }}
              </a-tag>
              <span>{{ testResult.duration_ms }} ms</span>
            </div>
            <pre v-if="testResult.stdout" class="run-out">{{ testResult.stdout }}</pre>
            <pre v-if="testResult.stderr" class="run-err">{{ testResult.stderr }}</pre>
          </div>
        </a-form-item>
        <a-form-item label="工作目录">
          <a-input v-model:value="form.work_dir" placeholder="请输入工作目录，留空则使用用户主目录" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.self-card {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px;
  margin-bottom: 16px;
  background: var(--ad-color-bg-layout);
}

.self-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--ad-color-text);
}

.self-path {
  margin-top: 6px;
  font-size: 12px;
  color: var(--ad-color-text-secondary);
  word-break: break-all;
}

.cmd-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.run-box {
  padding: 10px 12px;
  background: var(--ad-color-bg-layout);
  font-size: 12px;
}

.run-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
  color: var(--ad-color-text-secondary);
}

.run-out,
.run-err {
  margin: 0;
  max-height: 120px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.run-err {
  color: #ff4d4f;
  margin-top: 6px;
}

.danger {
  color: #ff4d4f;
}

.muted {
  color: var(--ad-color-text-secondary);
}
</style>
