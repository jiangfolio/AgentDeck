<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Modal, message } from 'antdv-next'
import {
  termHostGroupRemoveApi,
  termHostGroupSaveApi,
  termHostGroupsApi,
  termHostListApi,
  termHostRemoveApi,
  termHostSaveApi,
  termHostTestApi,
  type TermHostTestResult,
  type TermHost,
  type TermHostGroup,
} from '@/api/agent'
import { requestOpenSession } from '@/composables/useTerminalBridge'
import { useTableScrollY } from '@/composables/useTableScrollY'

const emit = defineEmits<{ switchTerminal: [] }>()

const loading = ref(false)
const items = ref<TermHost[]>([])
const total = ref(0)
const selected = ref<string[]>([])
const groups = ref<TermHostGroup[]>([])
const query = reactive({ page: 1, page_size: 20, keyword: '', group_id: undefined as string | undefined })
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef

const formOpen = ref(false)
const saving = ref(false)
const testing = ref(false)
const testResult = ref<TermHostTestResult | null>(null)
const form = reactive({
  id: '',
  group_id: undefined as string | undefined,
  title: '',
  host: '',
  port: 22,
  username: 'root',
  auth_type: 'password' as 'password' | 'private_key',
  secret: '',
  passphrase: '',
  description: '',
  sort: 0,
})

const groupOpen = ref(false)
const groupForm = reactive({ id: '', name: '', sort: 0 })

const columns = [
  { title: '主机地址', key: 'addr', width: 180, ellipsis: true },
  { title: '用户名', dataIndex: 'username', key: 'username', width: 110 },
  { title: '端口', dataIndex: 'port', key: 'port', width: 70 },
  { title: '分组', dataIndex: 'group_name', key: 'group_name', width: 100 },
  { title: '标题', dataIndex: 'title', key: 'title', width: 140, ellipsis: true },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '操作', key: 'op', width: 140 },
]

async function loadGroups() {
  groups.value = (await termHostGroupsApi()) || []
}

async function load() {
  loading.value = true
  try {
    const data = await termHostListApi({
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
    title: '',
    host: '',
    port: 22,
    username: 'root',
    auth_type: 'password',
    secret: '',
    passphrase: '',
    description: '',
    sort: 0,
  })
  testResult.value = null
  formOpen.value = true
}

function openEdit(row: TermHost) {
  Object.assign(form, {
    id: row.id,
    group_id: row.group_id || undefined,
    title: row.title,
    host: row.host,
    port: row.port,
    username: row.username,
    auth_type: row.auth_type,
    secret: '',
    passphrase: '',
    description: row.description,
    sort: row.sort,
  })
  testResult.value = null
  formOpen.value = true
}

async function testConnect() {
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await termHostTestApi({
      id: form.id || undefined,
      host: form.host,
      port: form.port,
      username: form.username,
      auth_type: form.auth_type,
      secret: form.secret || undefined,
      passphrase: form.passphrase || undefined,
    })
  } catch (e: any) {
    testResult.value = { ok: false, latency_ms: 0, message: e?.message || '测试失败' }
  } finally {
    testing.value = false
  }
}

async function submit() {
  saving.value = true
  try {
    await termHostSaveApi({
      id: form.id || undefined,
      group_id: form.group_id || '',
      title: form.title,
      host: form.host,
      port: form.port,
      username: form.username,
      auth_type: form.auth_type,
      secret: form.secret || undefined,
      passphrase: form.passphrase || undefined,
      description: form.description,
      sort: form.sort,
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
    title: '删除主机？',
    content: `将删除 ${ids.length} 台主机，不可恢复。`,
    onOk: async () => {
      await termHostRemoveApi(ids)
      message.success('已删除')
      await load()
    },
  })
}

function connect(row: TermHost) {
  requestOpenSession({
    kind: 'ssh',
    hostId: row.id,
    title: row.title || `${row.username}@${row.host}`,
  })
  emit('switchTerminal')
}

function openGroupMgr() {
  groupForm.id = ''
  groupForm.name = ''
  groupForm.sort = 0
  groupOpen.value = true
}

async function saveGroup() {
  try {
    await termHostGroupSaveApi({
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

function editGroup(g: TermHostGroup) {
  groupForm.id = g.id
  groupForm.name = g.name
  groupForm.sort = g.sort
}

function removeGroup(g: TermHostGroup) {
  Modal.confirm({
    title: '删除分组？',
    content: g.name,
    onOk: async () => {
      await termHostGroupRemoveApi(g.id)
      message.success('已删除')
      await loadGroups()
      if (query.group_id === g.id) query.group_id = undefined
      await load()
    },
  })
}

onMounted(async () => {
  await loadGroups()
  await load()
})
</script>

<template>
  <div ref="wrapRef" class="ad-page-fill host-page">
    <div class="ad-page-toolbar">
      <a-space>
        <a-button type="primary" @click="openCreate">添加主机</a-button>
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
          placeholder="搜索地址、标题或用户名"
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
          <template v-if="column.key === 'addr'">{{ record.host }}</template>
          <template v-else-if="column.key === 'op'">
            <a-space>
              <a @click="connect(record)">连接</a>
              <a @click="openEdit(record)">编辑</a>
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
      :title="form.id ? '编辑主机' : '添加主机'"
      :confirm-loading="saving"
      centered
      width="560px"
      :styles="{
        wrapper: { overflow: 'hidden' },
        body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' },
      }"
      destroy-on-hidden
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="标题" required>
          <a-input v-model:value="form.title" placeholder="请输入标题，如 生产机" />
        </a-form-item>
        <a-form-item label="地址" required>
          <a-input v-model:value="form.host" placeholder="请输入主机地址或 IP" />
        </a-form-item>
        <a-space style="width: 100%" :size="12">
          <a-form-item label="端口" required>
            <a-input-number v-model:value="form.port" :min="1" :max="65535" />
          </a-form-item>
          <a-form-item label="用户名" required>
            <a-input v-model:value="form.username" placeholder="请输入用户名" />
          </a-form-item>
        </a-space>
        <a-form-item label="分组">
          <a-select
            v-model:value="form.group_id"
            allow-clear
            placeholder="请选择分组，可不选"
            :options="groups.map((g) => ({ value: g.id, label: g.name }))"
          />
        </a-form-item>
        <a-form-item label="认证方式">
          <a-radio-group v-model:value="form.auth_type">
            <a-radio value="password">密码</a-radio>
            <a-radio value="private_key">私钥</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item :label="form.auth_type === 'password' ? '密码' : '私钥'">
          <a-textarea
            v-if="form.auth_type === 'private_key'"
            v-model:value="form.secret"
            :rows="4"
            :placeholder="form.id ? '留空则不修改' : '粘贴 PEM 私钥'"
          />
          <a-input-password
            v-else
            v-model:value="form.secret"
            :placeholder="form.id ? '留空则不修改' : '请输入密码'"
          />
        </a-form-item>
        <a-form-item v-if="form.auth_type === 'private_key'" label="私钥口令">
          <a-input-password v-model:value="form.passphrase" placeholder="可选；编辑时留空不改" />
        </a-form-item>
        <a-form-item label="描述">
          <a-input v-model:value="form.description" placeholder="请输入描述，可选" />
        </a-form-item>
      </a-form>
      <template #footer>
        <div class="host-footer">
          <span v-if="testResult" class="test-msg" :class="testResult.ok ? 'ok' : 'bad'">
            {{ testResult.message }}
          </span>
          <div class="host-footer-actions">
            <a-button :loading="testing" @click="testConnect">测试连接</a-button>
            <a-button @click="formOpen = false">取消</a-button>
            <a-button type="primary" :loading="saving" @click="submit">确定</a-button>
          </div>
        </div>
      </template>
    </a-modal>

    <a-modal
      v-model:open="groupOpen"
      title="主机分组"
      :footer="null"
      centered
      width="520px"
      :styles="{ body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' } }"
    >
      <a-space style="margin-bottom: 12px" wrap>
        <a-input v-model:value="groupForm.name" placeholder="分组名称" style="width: 180px" />
        <a-input-number v-model:value="groupForm.sort" placeholder="排序" />
        <a-button type="primary" @click="saveGroup">{{ groupForm.id ? '更新' : '添加' }}</a-button>
        <a-button v-if="groupForm.id" @click="groupForm.id = ''; groupForm.name = ''">取消编辑</a-button>
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
  </div>
</template>

<style scoped>
.host-page {
  padding: 0;
  height: 100%;
}
.ml-auto {
  margin-left: auto;
}
.danger {
  color: #ff4d4f;
}
.host-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  width: 100%;
}
.host-footer-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.test-msg {
  margin-right: auto;
  font-size: 13px;
  text-align: left;
}
.test-msg.ok {
  color: #389e0d;
}
.test-msg.bad {
  color: #cf1322;
}
</style>
