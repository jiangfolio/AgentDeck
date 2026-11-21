<script setup lang="ts">
import { terminalSettings } from '@/composables/useTerminalSettings'
</script>

<template>
  <div class="settings">
    <a-form layout="vertical" style="max-width: 420px">
      <a-form-item label="字体大小">
        <a-slider v-model:value="terminalSettings.fontSize" :min="10" :max="24" :marks="{ 12: '12', 14: '14', 16: '16', 20: '20' }" />
      </a-form-item>
      <a-form-item label="主题">
        <a-radio-group v-model:value="terminalSettings.theme">
          <a-radio-button value="dark">深色</a-radio-button>
          <a-radio-button value="light">浅色</a-radio-button>
        </a-radio-group>
      </a-form-item>
      <a-form-item label="SSH 主机指纹">
        <a-switch v-model:checked="terminalSettings.ignoreHostKey" checked-children="忽略校验" un-checked-children="拒绝未知" />
        <div class="hint">本机私用面板默认忽略指纹；关闭后当前版本将拒绝无法校验的主机。</div>
      </a-form-item>
    </a-form>
  </div>
</template>

<style scoped>
.settings {
  padding-top: 8px;
}
.hint {
  margin-top: 8px;
  color: var(--ad-color-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}
</style>
