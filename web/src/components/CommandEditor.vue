<script setup lang="ts">
import { computed } from 'vue'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'

const props = withDefaults(
  defineProps<{
    modelValue: string
    language?: string
    height?: string
    readonly?: boolean
  }>(),
  {
    language: 'shell',
    height: '180px',
    readonly: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const code = computed({
  get: () => props.modelValue,
  set: (v: string) => emit('update:modelValue', v ?? ''),
})

const options = computed(() => ({
  automaticLayout: true,
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: 'on' as const,
  scrollBeyondLastLine: false,
  wordWrap: 'on' as const,
  tabSize: 2,
  readOnly: props.readonly,
  renderLineHighlight: 'line' as const,
  padding: { top: 8, bottom: 8 },
}))
</script>

<template>
  <div class="command-editor" :style="{ height }">
    <VueMonacoEditor
      v-model:value="code"
      theme="vs"
      :language="language"
      :options="options"
    />
  </div>
</template>

<style scoped>
.command-editor {
  width: 100%;
  height: v-bind(height);
  min-height: 180px;
  max-height: v-bind(height);
  border: 1px solid var(--ad-color-border-secondary, #f0f0f0);
  overflow: hidden;
  background: #fff;
  box-sizing: border-box;
}
</style>
