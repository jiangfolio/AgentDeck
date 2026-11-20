<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message } from 'antdv-next'
import { diskListApi, type DiskItem } from '@/api/agent'

const loading = ref(false)
const items = ref<DiskItem[]>([])

function formatBytes(n: number) {
  if (!n) return '0B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)}${u[i]}`
}

async function load() {
  loading.value = true
  try {
    const data = await diskListApi()
    items.value = data.items || []
  } catch (e: any) {
    message.error(e?.message || '加载磁盘失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="ad-page-scroll disk-page">
    <div class="ad-page-toolbar">
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>
    <a-spin :spinning="loading">
      <a-empty v-if="!items.length" description="没有读到磁盘" />
      <div v-for="disk in items" :key="disk.name" class="disk-card">
        <div class="head">
          <span class="title">磁盘名称: {{ disk.name }}</span>
          <a-tag v-if="disk.system" color="orange">系统磁盘</a-tag>
          <span class="meta">大小: {{ formatBytes(disk.size_bytes) }}　分区: {{ disk.partitions?.length || 0 }}</span>
        </div>
        <a-table
          size="small"
          :pagination="false"
          row-key="name"
          :data-source="disk.partitions || []"
          :columns="[
            { title: '分区名称', dataIndex: 'name', key: 'name', width: 140 },
            { title: '大小', key: 'size', width: 100 },
            { title: '已用', key: 'used', width: 100 },
            { title: '可用', key: 'free', width: 100 },
            { title: '使用率', key: 'usage', width: 180 },
            { title: '挂载目录', dataIndex: 'mountpoint', key: 'mountpoint' },
            { title: '文件系统', dataIndex: 'fstype', key: 'fstype', width: 120 },
            { title: '操作', key: 'action', width: 120 },
          ]"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'size'">{{ formatBytes(record.size_bytes) }}</template>
            <template v-else-if="column.key === 'used'">{{ formatBytes(record.used_bytes) }}</template>
            <template v-else-if="column.key === 'free'">{{ formatBytes(record.free_bytes) }}</template>
            <template v-else-if="column.key === 'usage'">
              <a-progress :percent="Number((record.usage_percent || 0).toFixed(1))" size="small" />
            </template>
            <template v-else-if="column.key === 'action'">
              <span class="muted">无法操作</span>
            </template>
          </template>
        </a-table>
        <div v-if="disk.system" class="hint">提示：当前磁盘为系统盘，无法进行操作</div>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.disk-page {
  background: var(--ad-color-bg-container);
  padding: 16px;
}

.disk-card {
  margin-bottom: 16px;
  border: 1px solid var(--ad-color-border-secondary);
  padding: 12px 16px 8px;
}

.head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.title {
  font-weight: 600;
}

.meta {
  margin-left: auto;
  color: var(--ad-color-text-secondary);
  font-size: 13px;
}

.hint {
  margin-top: 8px;
  color: var(--ad-color-text-secondary);
  font-size: 12px;
}

.muted {
  color: var(--ad-color-text-secondary);
}
</style>
