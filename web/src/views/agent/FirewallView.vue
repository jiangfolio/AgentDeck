<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Modal, message } from 'antdv-next'
import {
  firewallExportApi,
  firewallImportApi,
  firewallInstallOfflineApi,
  firewallInstallOnlineApi,
  firewallRemoveApi,
  firewallResetApi,
  firewallRulesApi,
  firewallSaveApi,
  firewallSetEnabledApi,
  firewallStatusApi,
  type FirewallRule,
  type FirewallStatus,
} from '@/api/agent'

const loading = ref(false)
const status = ref<FirewallStatus | null>(null)
const items = ref<FirewallRule[]>([])
const total = ref(0)
const hint = ref('')
const query = reactive({ keyword: '', page: 1, page_size: 20 })

const formOpen = ref(false)
const saving = ref(false)
const form = reactive({
  id: '',
  action: 'allow',
  protocol: 'tcp',
  ip: '',
  port: '',
  description: '',
})

const importOpen = ref(false)
const importMode = ref<'merge' | 'replace'>('merge')
const importRules = ref<FirewallRule[]>([])

const installOpen = ref(false)
const installing = ref(false)
const installMode = ref<'online' | 'offline'>('online')
const pkgFile = ref<File | null>(null)

const needInstall = computed(() => !!status.value && !status.value.installed)
const canInstall = computed(() => !!status.value && (status.value.can_online || status.value.can_offline))
const installOkProps = computed(() => ({
  disabled: installMode.value === 'offline' && !pkgFile.value,
}))

async function loadStatus() {
  status.value = await firewallStatusApi()
}

async function load() {
  loading.value = true
  try {
    const [list] = await Promise.all([
      firewallRulesApi({
        keyword: query.keyword || undefined,
        page: query.page,
        page_size: query.page_size,
      }),
      loadStatus(),
    ])
    items.value = list.items || []
    total.value = list.total || 0
    hint.value = list.message || ''
  } catch (e: any) {
    message.error(e?.message || '加载防火墙失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  query.page = 1
  void load()
}

function openInstall() {
  const st = status.value
  if (!st) return
  installMode.value = st.can_online ? 'online' : 'offline'
  pkgFile.value = null
  installOpen.value = true
}

function openCreate() {
  if (needInstall.value) {
    openInstall()
    return
  }
  Object.assign(form, { id: '', action: 'allow', protocol: 'tcp', ip: '', port: '', description: '' })
  formOpen.value = true
}

function pickPkg(file: File) {
  const exts = status.value?.offline_ext || []
  const name = file.name.toLowerCase()
  if (!exts.some((ext) => name.endsWith(ext))) {
    message.error(exts.length ? `只接受 ${exts.join('、')} 安装包` : '不支持的安装包类型')
    return false
  }
  if (file.size <= 0 || file.size > 64 * 1024 * 1024) {
    message.error('安装包大小无效')
    return false
  }
  pkgFile.value = file
  return false
}

/**
 * 按弹窗里选的方式安装防火墙组件，失败时留在弹窗里。
 */
async function submitInstall() {
  // 1. 在线安装走包管理器；离线必须先选好安装包
  if (installMode.value === 'online') {
    await runInstall(() => firewallInstallOnlineApi())
    return
  }
  if (!pkgFile.value) {
    message.error('请选择安装包')
    throw new Error('请选择安装包')
  }
  const file = pkgFile.value
  // 2. 上传后由服务端核对文件头并安装
  await runInstall(() => firewallInstallOfflineApi(file))
}

async function runInstall(task: () => Promise<unknown>) {
  installing.value = true
  try {
    await task()
    message.success('已安装')
    installOpen.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message || '安装失败')
    throw e
  } finally {
    installing.value = false
  }
}

function openEdit(row: FirewallRule) {
  Object.assign(form, {
    id: row.id,
    action: row.action,
    protocol: row.protocol || 'any',
    ip: row.ip,
    port: row.port,
    description: row.description,
  })
  formOpen.value = true
}

async function onSave() {
  saving.value = true
  try {
    await firewallSaveApi({ ...form })
    message.success('已保存')
    formOpen.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function onToggle(enabled: boolean) {
  const run = async () => {
    try {
      await firewallSetEnabledApi(enabled)
      message.success(enabled ? '已开启' : '已关闭')
      await load()
    } catch (e: any) {
      message.error(e?.message || '操作失败')
    }
  }
  if (enabled) {
    void run()
    return
  }
  Modal.confirm({
    title: '关闭防火墙？',
    content: '关闭后入站规则不再生效。',
    okType: 'danger',
    onOk: run,
  })
}

function onRemove(row: FirewallRule) {
  Modal.confirm({
    title: `删除规则 ${row.port || row.id}？`,
    okType: 'danger',
    onOk: async () => {
      try {
        await firewallRemoveApi(row.id)
        message.success('已删除')
        await load()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

function onReset() {
  Modal.confirm({
    title: '重置防火墙？',
    content: '将恢复系统默认策略。此操作不可撤销。',
    okType: 'danger',
    onOk: () =>
      new Promise<void>((resolve, reject) => {
        Modal.confirm({
          title: '再次确认重置防火墙',
          okType: 'danger',
          onOk: async () => {
            try {
              await firewallResetApi()
              message.success('已重置')
              await load()
              resolve()
            } catch (e: any) {
              message.error(e?.message || '重置失败')
              reject(e)
            }
          },
          onCancel: () => resolve(),
        })
      }),
  })
}

async function onExport() {
  try {
    const rules = await firewallExportApi()
    const blob = new Blob([JSON.stringify(rules || [], null, 2)], { type: 'application/json' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = 'firewall-rules.json'
    a.click()
    URL.revokeObjectURL(a.href)
  } catch (e: any) {
    message.error(e?.message || '导出失败')
  }
}

function onPickImport(file: File) {
  const reader = new FileReader()
  reader.onload = () => {
    try {
      const parsed = JSON.parse(String(reader.result || '[]'))
      if (!Array.isArray(parsed)) throw new Error('格式无效')
      importRules.value = parsed
      importMode.value = 'merge'
      importOpen.value = true
    } catch {
      message.error('规则文件不是 JSON 数组')
    }
  }
  reader.readAsText(file)
  return false
}

async function onImport() {
  try {
    await firewallImportApi(importMode.value, importRules.value)
    message.success('已导入')
    importOpen.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message || '导入失败')
  }
}

function confirmImport() {
  if (importMode.value === 'replace') {
    Modal.confirm({
      title: '替换导入？',
      content: '只覆盖 AgentDeck 创建的规则，系统原有规则会保留。',
      onOk: onImport,
    })
    return
  }
  void onImport()
}

onMounted(load)
</script>

<template>
  <div class="ad-page-fill">
    <div class="status-bar">
      <a-tag color="blue">{{ status?.backend || '-' }}</a-tag>
      <a-tag v-if="needInstall" color="warning">未安装</a-tag>
      <a-tag v-else :color="status?.enabled ? 'success' : 'default'">{{ status?.enabled ? '已启用' : '未启用' }}</a-tag>
      <span class="ver">{{ status?.version || '-' }}</span>
      <a-switch
        :checked="!!status?.enabled"
        :disabled="needInstall"
        checked-children="开"
        un-checked-children="关"
        @change="(v: boolean) => onToggle(v)"
      />
      <a-button danger :disabled="needInstall" @click="onReset">重置</a-button>
    </div>
    <a-alert v-if="needInstall" type="warning" show-icon class="fw-alert">
      <template #message>防火墙组件未安装</template>
      <template #description>
        <div>{{ status?.message }}</div>
        <a-button v-if="canInstall" type="primary" class="install-btn" @click="openInstall">安装</a-button>
      </template>
    </a-alert>
    <div v-else-if="hint" class="hint">{{ hint }}</div>

    <div class="ad-page-toolbar">
      <a-button type="primary" :disabled="needInstall" @click="openCreate">创建</a-button>
      <a-upload :before-upload="onPickImport" :show-upload-list="false" accept="application/json,.json" :disabled="needInstall">
        <a-button :disabled="needInstall">导入</a-button>
      </a-upload>
      <a-button :disabled="needInstall" @click="onExport">导出</a-button>
      <a-input
        v-model:value="query.keyword"
        allow-clear
        placeholder="搜索"
        style="width: 200px; margin-left: auto"
        @press-enter="onSearch"
      />
      <a-button @click="onSearch">搜索</a-button>
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>

    <div class="ad-page-table">
      <a-table
        row-key="id"
        size="middle"
        :loading="loading"
        :pagination="false"
        :data-source="items"
        :columns="[
          { title: '动作', key: 'action', width: 90 },
          { title: '优先级', dataIndex: 'priority', key: 'priority', width: 90 },
          { title: '协议', dataIndex: 'protocol', key: 'protocol', width: 90 },
          { title: 'IP', key: 'ip' },
          { title: '端口', dataIndex: 'port', key: 'port', width: 120 },
          { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
          { title: '操作', key: 'op', width: 140 },
        ]"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <a-tag :color="record.action === 'allow' ? 'success' : 'error'">
              {{ record.action === 'allow' ? '允许' : '拒绝' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'ip'">{{ record.ip || '0.0.0.0/0（任何位置）' }}</template>
          <template v-else-if="column.key === 'op'">
            <a-space>
              <a @click="openEdit(record)">编辑</a>
              <a class="danger" @click="onRemove(record)">删除</a>
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
      :title="form.id ? '编辑规则' : '创建规则'"
      :confirm-loading="saving"
      centered
      @ok="onSave"
    >
      <a-form layout="vertical">
        <a-form-item label="动作" required>
          <a-select v-model:value="form.action" :options="[{ value: 'allow', label: '允许' }, { value: 'deny', label: '拒绝' }]" />
        </a-form-item>
        <a-form-item label="协议" required>
          <a-select v-model:value="form.protocol" :options="[{ value: 'tcp', label: 'TCP' }, { value: 'udp', label: 'UDP' }, { value: 'any', label: '任意' }]" />
        </a-form-item>
        <a-form-item label="来源 IP">
          <a-input v-model:value="form.ip" placeholder="留空表示任意，或填写 IP/CIDR" />
        </a-form-item>
        <a-form-item label="端口">
          <a-input v-model:value="form.port" placeholder="如 80 或 80:90" />
        </a-form-item>
        <a-form-item label="描述">
          <a-input v-model:value="form.description" maxlength="128" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="installOpen"
      title="安装防火墙"
      ok-text="安装"
      :confirm-loading="installing"
      :ok-button-props="installOkProps"
      centered
      @ok="submitInstall"
    >
      <a-radio-group
        v-if="status?.can_online && status?.can_offline"
        v-model:value="installMode"
        class="install-mode"
      >
        <a-radio value="online">在线安装</a-radio>
        <a-radio value="offline">离线安装</a-radio>
      </a-radio-group>
      <p v-if="installMode === 'online'" class="install-tip">
        使用 {{ status?.manager || '包管理器' }} 从软件源安装 {{ status?.package || 'ufw' }}，需要管理员权限和网络，可能需要几分钟。
      </p>
      <div v-else>
        <p class="install-tip">上传 {{ (status?.offline_ext || []).join('、') || '安装包' }}。只安装防火墙组件，不会执行安装包以外的命令。</p>
        <a-upload :before-upload="pickPkg" :show-upload-list="false" :accept="(status?.offline_ext || []).join(',')">
          <a-button>选择安装包</a-button>
        </a-upload>
        <p v-if="pkgFile" class="pkg-name">{{ pkgFile.name }}</p>
      </div>
    </a-modal>

    <a-modal v-model:open="importOpen" title="导入规则" centered @ok="confirmImport">
      <p>共 {{ importRules.length }} 条</p>
      <a-radio-group v-model:value="importMode">
        <a-radio value="merge">合并（跳过已存在的同类规则）</a-radio>
        <a-radio value="replace">替换 AgentDeck 创建的规则</a-radio>
      </a-radio-group>
    </a-modal>
  </div>
</template>

<style scoped>
.status-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.ver {
  color: var(--ad-color-text-secondary);
  font-size: 13px;
}

.hint {
  flex-shrink: 0;
  margin: -4px 0 12px;
  color: var(--ad-color-text-secondary);
  font-size: 12px;
}

.fw-alert {
  flex-shrink: 0;
  margin-bottom: 12px;
}

.install-btn {
  margin-top: 8px;
}

.install-mode {
  margin-bottom: 12px;
}

.install-tip {
  margin: 0 0 12px;
  color: var(--ad-color-text-secondary);
}

.pkg-name {
  margin: 8px 0 0;
}

.danger {
  color: #ff4d4f;
}
</style>
