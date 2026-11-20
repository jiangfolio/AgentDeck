<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Modal, message } from 'antdv-next'
import {
  HELLO_COMMAND,
  commandRunApi,
  cronListApi,
  cronRemoveApi,
  cronRunsApi,
  cronSaveApi,
  cronSetEnabledApi,
  type CommandRunResult,
  type CronJob,
  type CronRun,
} from '@/api/agent'
import CommandEditor from '@/components/CommandEditor.vue'
import { useTableScrollY } from '@/composables/useTableScrollY'

const loading = ref(false)
const items = ref<CronJob[]>([])
const total = ref(0)
const query = reactive({ page: 1, page_size: 20 })

const formOpen = ref(false)
const saving = ref(false)
const testing = ref(false)
const testResult = ref<CommandRunResult | null>(null)
const form = reactive({
  id: '',
  name: '',
  expr: '',
  command: '',
  work_dir: '',
  enabled: true,
})

const runsOpen = ref(false)
const runsLoading = ref(false)
const runs = ref<CronRun[]>([])
const runsTitle = ref('')
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 160 },
  { title: '表达式', dataIndex: 'expr', key: 'expr', width: 140 },
  { title: '命令', dataIndex: 'command', key: 'command', ellipsis: true },
  { title: '工作目录', dataIndex: 'work_dir', key: 'work_dir', width: 160, ellipsis: true },
  { title: '启用', key: 'enabled', width: 90 },
  { title: '下次运行', key: 'next_run_at', width: 180 },
  { title: '操作', key: 'action', width: 200 },
]

async function load() {
  loading.value = true
  try {
    const data = await cronListApi({ page: query.page, page_size: query.page_size })
    items.value = data.items || []
    total.value = Number(data.total || 0)
    recalc()
  } catch (e: any) {
    message.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, {
    id: '',
    name: '',
    expr: '',
    command: '',
    work_dir: '',
    enabled: true,
  })
  testResult.value = null
  formOpen.value = true
}

function openEdit(row: CronJob) {
  Object.assign(form, {
    id: row.id,
    name: row.name,
    expr: row.expr,
    command: row.command,
    work_dir: row.work_dir,
    enabled: row.enabled,
  })
  testResult.value = null
  formOpen.value = true
}

function fillHello() {
  form.command = HELLO_COMMAND
  if (!form.name.trim()) form.name = 'Hello 测试'
  if (!form.expr.trim()) form.expr = '*/1 * * * *'
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
  if (!form.name.trim() || !form.expr.trim() || !form.command.trim()) {
    message.warning('请填写名称、表达式与命令')
    return
  }
  saving.value = true
  try {
    await cronSaveApi({
      id: form.id || undefined,
      name: form.name.trim(),
      expr: form.expr.trim(),
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

async function toggleEnabled(row: CronJob, enabled: boolean) {
  try {
    await cronSetEnabledApi(row.id, enabled)
    message.success(enabled ? '已启用' : '已禁用')
    await load()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  }
}

function confirmRemove(row: CronJob) {
  Modal.confirm({
    title: `删除定时任务「${row.name}」？`,
    okType: 'danger',
    onOk: async () => {
      try {
        await cronRemoveApi(row.id)
        message.success('已删除')
        await load()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

async function openRuns(row: CronJob) {
  runsTitle.value = `运行记录 · ${row.name}`
  runsOpen.value = true
  runsLoading.value = true
  runs.value = []
  try {
    runs.value = (await cronRunsApi(row.id)) || []
  } catch (e: any) {
    message.error(e?.message || '加载运行记录失败')
  } finally {
    runsLoading.value = false
  }
}

function formatNext(ts: number) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
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
    <div class="ad-page-toolbar">
      <a-button type="primary" @click="openCreate">新建任务</a-button>
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
          <template v-if="column.key === 'expr'">
            <a-tag color="blue">{{ record.expr }}</a-tag>
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-switch
              :checked="record.enabled"
              @change="(v: boolean) => toggleEnabled(record, v)"
            />
          </template>
          <template v-else-if="column.key === 'next_run_at'">
            {{ formatNext(record.next_run_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a @click="openEdit(record)">编辑</a>
              <a @click="openRuns(record)">记录</a>
              <a class="danger" @click="confirmRemove(record)">删除</a>
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
      v-model:open="formOpen"
      :title="form.id ? '编辑定时任务' : '新建定时任务'"
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
          <a-input v-model:value="form.name" maxlength="128" placeholder="请输入任务名称" />
        </a-form-item>
        <a-form-item label="Cron 表达式（五段）" required>
          <a-input v-model:value="form.expr" placeholder="请输入 Cron 表达式，如 */1 * * * *" />
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

    <a-modal
      v-model:open="runsOpen"
      :title="runsTitle"
      :footer="null"
      centered
      width="800px"
      :styles="{
        wrapper: { overflow: 'hidden' },
        body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' },
      }"
    >
      <a-spin :spinning="runsLoading">
        <a-table
          row-key="ts"
          size="small"
          :pagination="false"
          :data-source="runs"
          :columns="[
            { title: '时间', dataIndex: 'ts', key: 'ts', width: 200 },
            { title: '退出码', key: 'exit_code', width: 100 },
            { title: '耗时(ms)', dataIndex: 'duration_ms', key: 'duration_ms', width: 100 },
            { title: 'stdout', dataIndex: 'stdout', key: 'stdout', ellipsis: true },
            { title: 'stderr', dataIndex: 'stderr', key: 'stderr', ellipsis: true },
          ]"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'exit_code'">
              <a-tag :color="record.exit_code === 0 ? 'success' : 'error'">
                {{ record.exit_code }}
              </a-tag>
            </template>
          </template>
        </a-table>
      </a-spin>
    </a-modal>
  </div>
</template>

<style scoped>
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
</style>
