<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Modal, message } from 'antdv-next'
import {
  scriptGroupRemoveApi,
  scriptGroupSaveApi,
  scriptGroupsApi,
  scriptListApi,
  scriptRemoveApi,
  scriptRunApi,
  scriptSaveApi,
  termHostListApi,
  type ScriptGroup,
  type ScriptItem,
  type ScriptRunResult,
  type TermHost,
} from '@/api/agent'
import CommandEditor from '@/components/CommandEditor.vue'
import { requestOpenSession } from '@/composables/useTerminalBridge'
import { useTableScrollY } from '@/composables/useTableScrollY'
import { useRouter } from 'vue-router'

const router = useRouter()
const loading = ref(false)
const items = ref<ScriptItem[]>([])
const total = ref(0)
const selected = ref<string[]>([])
const groups = ref<ScriptGroup[]>([])
const query = reactive({ page: 1, page_size: 20, keyword: '', group_id: undefined as string | undefined })
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef

const formOpen = ref(false)
const saving = ref(false)
const form = reactive({
  id: '',
  group_id: undefined as string | undefined,
  name: '',
  content: '',
  interactive: false,
  description: '',
})

const groupOpen = ref(false)
const groupForm = reactive({ id: '', name: '', sort: 0 })

const runOpen = ref(false)
const runTarget = ref<ScriptItem | null>(null)
const runHostId = ref<string | undefined>(undefined)
const hosts = ref<TermHost[]>([])
const runLoading = ref(false)
const runResult = ref<ScriptRunResult | null>(null)

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 180, ellipsis: true },
  { title: '交互式', key: 'interactive', width: 80 },
  { title: '分组', dataIndex: 'group_name', key: 'group_name', width: 100 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '时间', key: 'time', width: 170 },
  { title: '操作', key: 'op', width: 220 },
]

async function loadGroups() {
  groups.value = (await scriptGroupsApi()) || []
}

async function load() {
  loading.value = true
  try {
    const data = await scriptListApi({
      page: query.page,
      page_size: query.page_size,
      keyword: query.keyword || undefined,
      group_id: query.group_id || undefined,
    })
    items.value = data.items || []
    total.value = Number(data.total || 0)
    selected.value = []
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
    group_id: undefined,
    name: '',
    content: '',
    interactive: false,
    description: '',
  })
  formOpen.value = true
}

function openEdit(row: ScriptItem) {
  Object.assign(form, {
    id: row.id,
    group_id: row.group_id || undefined,
    name: row.name,
    content: row.content,
    interactive: row.interactive,
    description: row.description,
  })
  formOpen.value = true
}

function openClone(row: ScriptItem) {
  Object.assign(form, {
    id: '',
    group_id: row.group_id || undefined,
    name: `${row.name} 副本`,
    content: row.content,
    interactive: row.interactive,
    description: row.description,
  })
  formOpen.value = true
}

async function submit() {
  saving.value = true
  try {
    await scriptSaveApi({
      id: form.id || undefined,
      group_id: form.group_id || '',
      name: form.name,
      content: form.content,
      interactive: form.interactive,
      description: form.description,
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

function onRemove(ids: string[]) {
  Modal.confirm({
    title: '删除脚本？',
    content: `将删除 ${ids.length} 条，不可恢复。`,
    onOk: async () => {
      await scriptRemoveApi(ids)
      message.success('已删除')
      await load()
    },
  })
}

async function openRun(row: ScriptItem) {
  runTarget.value = row
  runHostId.value = undefined
  runResult.value = null
  if (row.interactive) {
    // 交互式：直接开本机终端并注入
    requestOpenSession({ kind: 'local', title: `脚本·${row.name}`, inject: row.content })
    message.info('已在终端打开会话并注入脚本')
    router.push({ name: 'AgentTerminal' })
    return
  }
  try {
    const data = await termHostListApi({ page: 1, page_size: 100 })
    hosts.value = data.items || []
  } catch {
    hosts.value = []
  }
  runOpen.value = true
}

async function doRun() {
  if (!runTarget.value) return
  runLoading.value = true
  runResult.value = null
  try {
    const res = await scriptRunApi({
      id: runTarget.value.id,
      host_id: runHostId.value || undefined,
    })
    if (res.interactive && res.content) {
      requestOpenSession({
        kind: runHostId.value ? 'ssh' : 'local',
        hostId: runHostId.value,
        title: `脚本·${runTarget.value.name}`,
        inject: res.content,
      })
      runOpen.value = false
      router.push({ name: 'AgentTerminal' })
      return
    }
    runResult.value = res
  } catch (e: any) {
    message.error(e?.message || '执行失败')
  } finally {
    runLoading.value = false
  }
}

function openGroupMgr() {
  groupForm.id = ''
  groupForm.name = ''
  groupForm.sort = 0
  groupOpen.value = true
}

async function saveGroup() {
  try {
    await scriptGroupSaveApi({
      id: groupForm.id || undefined,
      name: groupForm.name,
      sort: groupForm.sort,
    })
    message.success('分组已保存')
    groupForm.id = ''
    groupForm.name = ''
    await loadGroups()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  }
}

function editGroup(g: ScriptGroup) {
  groupForm.id = g.id
  groupForm.name = g.name
  groupForm.sort = g.sort
}

function removeGroup(g: ScriptGroup) {
  Modal.confirm({
    title: '删除分组？',
    content: g.name,
    onOk: async () => {
      await scriptGroupRemoveApi(g.id)
      message.success('已删除')
      await loadGroups()
      if (query.group_id === g.id) query.group_id = undefined
      await load()
    },
  })
}

function fmtTime(v: string) {
  if (!v) return '-'
  return String(v).replace('T', ' ').slice(0, 19)
}

onMounted(async () => {
  await loadGroups()
  await load()
})
</script>

<template>
  <div ref="wrapRef" class="ad-page-fill">
    <div class="ad-page-toolbar">
      <a-space>
        <a-button type="primary" @click="openCreate">创建</a-button>
        <a-button @click="openGroupMgr">分组</a-button>
        <a-button danger :disabled="!selected.length" @click="onRemove(selected)">删除</a-button>
      </a-space>
      <div class="ml-auto flex flex-wrap gap-2">
        <a-select
          v-model:value="query.group_id"
          allow-clear
          placeholder="请选择分组"
          style="width: 140px"
          :options="groups.map((g) => ({ value: g.id, label: g.name }))"
          @change="() => { query.page = 1; load() }"
        />
        <a-input-search
          v-model:value="query.keyword"
          placeholder="搜索名称或描述"
          style="width: 180px"
          allow-clear
          @search="() => { query.page = 1; load() }"
        />
        <a-button @click="load">刷新</a-button>
      </div>
    </div>
    <div class="ad-page-table">
      <a-table
        row-key="id"
        size="small"
        :loading="loading"
        :columns="columns"
        :data-source="items"
        :pagination="false"
        :scroll="{ y: scrollY }"
        :row-selection="{ selectedRowKeys: selected, onChange: (ks: (string|number)[]) => (selected = ks.map(String)) }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'interactive'">
            {{ record.interactive ? '是' : '否' }}
          </template>
          <template v-else-if="column.key === 'time'">
            {{ fmtTime(record.updated_at) }}
          </template>
          <template v-else-if="column.key === 'op'">
            <a-space>
              <a @click="openRun(record)">执行</a>
              <a @click="openClone(record)">克隆</a>
              <a @click="openEdit(record)">编辑</a>
              <a class="danger" @click="onRemove([record.id])">删除</a>
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
        @change="load"
      />
    </div>

    <a-modal
      v-model:open="formOpen"
      :title="form.id ? '编辑脚本' : '创建脚本'"
      width="760px"
      :confirm-loading="saving"
      centered
      :styles="{
        wrapper: { overflow: 'hidden' },
        body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' },
      }"
      destroy-on-hidden
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="form.name" maxlength="128" placeholder="请输入脚本名称" />
        </a-form-item>
        <a-form-item label="分组">
          <a-select
            v-model:value="form.group_id"
            allow-clear
            placeholder="请选择分组，可不选"
            :options="groups.map((g) => ({ value: g.id, label: g.name }))"
          />
        </a-form-item>
        <a-form-item label="交互式">
          <a-switch v-model:checked="form.interactive" />
          <span class="hint">开启后执行时打开终端并注入脚本</span>
        </a-form-item>
        <a-form-item label="描述">
          <a-input v-model:value="form.description" placeholder="请输入描述，可选" />
        </a-form-item>
        <a-form-item label="内容" required>
          <CommandEditor v-model="form.content" language="shell" height="280px" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="groupOpen"
      title="脚本分组"
      :footer="null"
      centered
      width="520px"
      :styles="{ body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' } }"
    >
      <a-space style="margin-bottom: 12px" wrap>
        <a-input v-model:value="groupForm.name" placeholder="分组名称" style="width: 180px" />
        <a-input-number v-model:value="groupForm.sort" />
        <a-button type="primary" @click="saveGroup">{{ groupForm.id ? '更新' : '添加' }}</a-button>
      </a-space>
      <a-list size="small" :data-source="groups" bordered>
        <template #renderItem="{ item }">
          <a-list-item>
            <span>{{ item.name }}</span>
            <template #actions>
              <a @click="editGroup(item)">编辑</a>
              <a class="danger" @click="removeGroup(item)">删除</a>
            </template>
          </a-list-item>
        </template>
      </a-list>
    </a-modal>

    <a-modal
      v-model:open="runOpen"
      title="执行脚本"
      :confirm-loading="runLoading"
      centered
      ok-text="执行"
      :styles="{ body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' } }"
      @ok="doRun"
    >
      <p class="hint">目标：{{ runTarget?.name }}</p>
      <a-form-item label="主机">
        <a-select
          v-model:value="runHostId"
          allow-clear
          placeholder="请选择主机，留空则在本机执行"
          style="width: 100%"
          :options="hosts.map((h) => ({ value: h.id, label: `${h.title} (${h.host})` }))"
        />
      </a-form-item>
      <pre v-if="runResult" class="run-out">退出码: {{ runResult.exit_code }}
耗时: {{ runResult.duration_ms }}ms

{{ runResult.stdout || '' }}
{{ runResult.stderr || '' }}</pre>
    </a-modal>
  </div>
</template>

<style scoped>
.ml-auto {
  margin-left: auto;
}
.danger {
  color: #ff4d4f;
}
.hint {
  margin-left: 8px;
  color: var(--ad-color-text-secondary);
  font-size: 12px;
}
.run-out {
  margin-top: 12px;
  max-height: 280px;
  overflow: auto;
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 4px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
