<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Modal, message } from 'antdv-next'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import { filesReadApi, filesWriteApi } from '@/api/agent'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{
  path: string
}>()

const loading = ref(false)
const saving = ref(false)
const title = ref('')
const language = ref('plaintext')
const content = ref('')
const original = ref('')
const dirty = computed(() => content.value !== original.value)

const options = computed(() => ({
  automaticLayout: true,
  minimap: { enabled: true, scale: 1 },
  fontSize: 13,
  lineNumbers: 'on' as const,
  scrollBeyondLastLine: false,
  wordWrap: 'on' as const,
  tabSize: 2,
  readOnly: false,
  renderLineHighlight: 'line' as const,
  padding: { top: 8, bottom: 8 },
}))

/**
 * 打开路径时拉取文本内容。
 */
async function loadFile() {
  // 1. 无路径则不请求
  if (!props.path) return
  loading.value = true
  try {
    // 2. 读文件并填充编辑器
    const data = await filesReadApi(props.path)
    title.value = data.name || props.path
    language.value = data.language || 'plaintext'
    content.value = data.content ?? ''
    original.value = content.value
  } catch (e: any) {
    message.error(e?.message || '打开失败')
    open.value = false
  } finally {
    loading.value = false
  }
}

async function onSave() {
  saving.value = true
  try {
    await filesWriteApi(props.path, content.value)
    original.value = content.value
    message.success('已保存')
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function requestClose() {
  if (!dirty.value) {
    open.value = false
    return
  }
  Modal.confirm({
    title: '有未保存的修改',
    content: '关闭将丢失未保存内容。',
    okText: '仍要关闭',
    okType: 'danger',
    onOk: () => {
      open.value = false
    },
  })
}

watch(
  () => open.value,
  (v) => {
    if (v) void loadFile()
  },
)
</script>

<template>
  <a-modal
    :open="open"
    :title="title || '编辑文件'"
    :footer="null"
    :mask-closable="false"
    :destroy-on-hidden="true"
    centered
    width="920px"
    :styles="{
      wrapper: { overflow: 'hidden' },
      body: { paddingTop: '12px' },
    }"
    @cancel="requestClose"
  >
    <a-spin :spinning="loading">
      <div class="editor-toolbar">
        <span class="path" :title="path">{{ path }}</span>
        <a-space>
          <a-tag v-if="dirty" color="orange">未保存</a-tag>
          <a-button :loading="saving" type="primary" @click="onSave">保存</a-button>
          <a-button @click="requestClose">关闭</a-button>
        </a-space>
      </div>
      <div class="editor-wrap">
        <VueMonacoEditor
          v-model:value="content"
          theme="vs"
          :language="language"
          :options="options"
        />
      </div>
    </a-spin>
  </a-modal>
</template>

<style scoped>
.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.path {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--ad-color-text-secondary);
}

.editor-wrap {
  height: min(68vh, 640px);
  border: 1px solid var(--ad-color-border-secondary, #f0f0f0);
  overflow: hidden;
}
</style>
