<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Modal, message } from 'antdv-next'
import {
  termCmdGroupsApi,
  termCmdListApi,
  termCmdRemoveApi,
  termCmdSaveApi,
  type TermQuickCmd,
} from '@/api/agent'
import { useTableScrollY } from '@/composables/useTableScrollY'

const emit = defineEmits<{ inject: [command: string]; switchTerminal: [] }>()

const loading = ref(false)
const items = ref<TermQuickCmd[]>([])
const total = ref(0)
const selected = ref<string[]>([])
const groupNames = ref<string[]>([])
const query = reactive({ page: 1, page_size: 20, keyword: '', group_name: undefined as string | undefined })
const { wrapRef, scrollY, recalc } = useTableScrollY()
void wrapRef

const formOpen = ref(false)
const saving = ref(false)
const form = reactive({ id: '', group_name: '', name: '', command: '', sort: 0 })

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 160 },
  { title: '分组', dataIndex: 'group_name', key: 'group_name', width: 120 },
  { title: '命令', dataIndex: 'command', key: 'command', ellipsis: true },
  { title: '操作', key: 'op', width: 160 },
]

async function loadGroups() {
  groupNames.value = (await termCmdGroupsApi()) || []
}

async function load() {
  loading.value = true
  try {
    const data = await termCmdListApi({
      page: query.page,
      page_size: query.page_size,
      keyword: query.keyword || undefined,
      group_name: query.group_name || undefined,
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
  Object.assign(form, { id: '', group_name: '', name: '', command: '', sort: 0 })
  formOpen.value = true
}

function openEdit(row: TermQuickCmd) {
  Object.assign(form, {
    id: row.id,
    group_name: row.group_name,
    name: row.name,
    command: row.command,
    sort: row.sort,
  })
  formOpen.value = true
}

async function submit() {
  saving.value = true
  try {
    await termCmdSaveApi({
      id: form.id || undefined,
      group_name: form.group_name,
      name: form.name,
      command: form.command,
      sort: form.sort,
    })
    message.success('已保存')
    formOpen.value = false
    await loadGroups()
    await load()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function onRemove(ids: string[]) {
  Modal.confirm({
    title: '删除快速命令？',
    onOk: async () => {
      await termCmdRemoveApi(ids)
      message.success('已删除')
      await load()
      await loadGroups()
    },
  })
}

function runCmd(row: TermQuickCmd) {
  emit('inject', row.command)
  emit('switchTerminal')
}

onMounted(async () => {
  await loadGroups()
  await load()
})
</script>

<template>
  <div ref="wrapRef" class="ad-page-fill cmd-page">
    <div class="ad-page-toolbar">
      <a-space>
        <a-button type="primary" @click="openCreate">创建</a-button>
        <a-button danger :disabled="!selected.length" @click="onRemove(selected)">删除</a-button>
      </a-space>
      <div class="ml-auto flex flex-wrap gap-2">
        <a-select
          v-model:value="query.group_name"
          allow-clear
          placeholder="请选择分组"
          style="width: 140px"
          :options="groupNames.map((g) => ({ value: g, label: g }))"
          @change="() => { query.page = 1; load() }"
        />
        <a-input-search
          v-model:value="query.keyword"
          placeholder="搜索名称或命令"
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
          <template v-if="column.key === 'op'">
            <a-space>
              <a @click="runCmd(record)">执行</a>
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
      :title="form.id ? '编辑命令' : '创建命令'"
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
        <a-form-item label="名称" required>
          <a-input v-model:value="form.name" placeholder="请输入命令名称" />
        </a-form-item>
        <a-form-item label="分组">
          <a-auto-complete
            v-model:value="form.group_name"
            :options="groupNames.map((g) => ({ value: g }))"
            placeholder="请输入或选择分组，可选"
          />
        </a-form-item>
        <a-form-item label="命令" required>
          <a-textarea v-model:value="form.command" :rows="4" placeholder="请输入要注入终端的命令" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.cmd-page {
  padding: 0;
  height: 100%;
}
.ml-auto {
  margin-left: auto;
}
.danger {
  color: #ff4d4f;
}
</style>
