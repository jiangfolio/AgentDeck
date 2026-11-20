<script setup lang="ts">
import { computed, ref } from 'vue'
import { message } from 'antdv-next'
import { hostOverviewApi } from '@/api/system'
import { usePolling } from '@/composables/usePolling'
import { useRealtime } from '@/composables/useRealtime'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const now = ref(new Date())

const { data, loading } = useRealtime((signal) => hostOverviewApi(signal), {
  key: 'system:host-overview',
  interval: 5000,
  onError: (e: unknown) => {
    const err = e as { message?: string }
    message.error(err?.message || '获取系统信息失败')
  },
})

usePolling(() => {
  now.value = new Date()
}, { interval: 1000 })

const displayName = computed(
  () => userStore.user?.display_name || userStore.user?.username || '用户',
)

const greetText = computed(() => {
  const h = now.value.getHours()
  if (h < 6) return '夜深了'
  if (h < 12) return '上午好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const clockText = computed(() => {
  const d = now.value
  const week = ['日', '一', '二', '三', '四', '五', '六'][d.getDay()]
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} 星期${week} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
})

function formatBytes(n: number) {
  if (!n || n <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let v = n
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

function formatUptime(sec: number) {
  const s = Math.max(0, Math.floor(sec || 0))
  const days = Math.floor(s / 86400)
  const hours = Math.floor((s % 86400) / 3600)
  const mins = Math.floor((s % 3600) / 60)
  if (days > 0) return `${days} 天 ${hours} 小时 ${mins} 分`
  if (hours > 0) return `${hours} 小时 ${mins} 分`
  return `${mins} 分`
}

function osLabel(os?: string) {
  const v = (os || '').toLowerCase()
  if (v === 'linux') return 'Linux'
  if (v === 'windows') return 'Windows'
  if (v === 'darwin') return 'macOS'
  return os || '-'
}

function usageColor(percent: number) {
  if (percent >= 90) return '#ff4d4f'
  if (percent >= 70) return '#faad14'
  return '#1677ff'
}
</script>

<template>
  <div class="home-page ad-page-scroll">
    <div class="welcome-card">
      <div class="welcome-left">
        <div class="welcome-greet">{{ greetText }}，{{ displayName }}</div>
        <div class="welcome-clock">{{ clockText }}</div>
      </div>
    </div>

    <a-spin :spinning="loading && !data">
      <div v-if="data" class="home-body">
        <div class="card-grid">
          <div class="info-card card-host">
            <div class="card-title">主机信息</div>
            <div class="card-body">
              <div class="host-badge">{{ osLabel(data.host.os) }} · {{ data.host.arch || '-' }}</div>
              <div class="kv"><span>主机名</span><b>{{ data.host.hostname || '-' }}</b></div>
              <div class="kv"><span>平台</span><b>{{ data.host.platform || '-' }} {{ data.host.platform_version }}</b></div>
              <div class="kv"><span>内核</span><b>{{ data.host.kernel_version || '-' }}</b></div>
              <div class="kv"><span>运行时长</span><b>{{ formatUptime(data.host.uptime_seconds) }}</b></div>
              <div class="host-foot">
                <div class="stat-block">
                  <div class="stat-label">进程 PID</div>
                  <div class="stat-value">{{ data.runtime.pid }}</div>
                </div>
                <div class="stat-block">
                  <div class="stat-label">Goroutine</div>
                  <div class="stat-value">{{ data.runtime.goroutines }}</div>
                </div>
                <div class="stat-block">
                  <div class="stat-label">Go</div>
                  <div class="stat-value small">{{ data.runtime.go_version }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="info-card card-cpu">
            <div class="card-title">CPU</div>
            <div class="card-body chart-body">
              <div class="chart-wrap">
                <a-progress
                  type="circle"
                  :percent="Number(data.cpu.usage_percent.toFixed(2))"
                  :size="148"
                  :stroke-color="usageColor(data.cpu.usage_percent)"
                  :stroke-width="10"
                />
              </div>
              <div class="chart-meta">
                <div class="meta-name" :title="data.cpu.model_name">{{ data.cpu.model_name || '-' }}</div>
                <div class="meta-row">
                  <span>物理核 {{ data.cpu.physical_cores }}</span>
                  <span>逻辑核 {{ data.cpu.logical_cores }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="info-card card-mem">
            <div class="card-title">内存</div>
            <div class="card-body chart-body">
              <div class="chart-wrap">
                <a-progress
                  type="dashboard"
                  :percent="Number(data.memory.usage_percent.toFixed(2))"
                  :size="148"
                  :gap-degree="70"
                  :stroke-color="usageColor(data.memory.usage_percent)"
                  :stroke-width="10"
                />
              </div>
              <div class="chart-meta">
                <div class="meta-row">
                  <span>总量 {{ formatBytes(data.memory.total_bytes) }}</span>
                </div>
                <div class="meta-row">
                  <span>已用 {{ formatBytes(data.memory.used_bytes) }}</span>
                  <span>可用 {{ formatBytes(data.memory.available_bytes) }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="info-card card-swap">
            <div class="card-title">交换区</div>
            <div class="card-body">
              <div class="swap-hero">
                <div class="swap-percent" :style="{ color: usageColor(data.swap.usage_percent) }">
                  {{ data.swap.total_bytes ? data.swap.usage_percent.toFixed(1) : '0.0' }}%
                </div>
                <div class="swap-sub">已用 {{ formatBytes(data.swap.used_bytes) }} / {{ formatBytes(data.swap.total_bytes) }}</div>
              </div>
              <a-progress
                type="line"
                :percent="Number((data.swap.usage_percent || 0).toFixed(2))"
                :stroke-color="{
                  '0%': '#1677ff',
                  '100%': usageColor(data.swap.usage_percent || 0),
                }"
                :show-info="false"
                :stroke-width="14"
              />
              <div class="swap-legend">
                <span>可用 {{ formatBytes(data.swap.available_bytes) }}</span>
                <span>逻辑 CPU {{ data.runtime.num_cpu }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="disk-section">
          <div class="disk-head">
            <div class="card-title">磁盘</div>
            <span class="disk-count">{{ data.disks.length }} 个分区</span>
          </div>
          <div v-if="data.disks.length" class="disk-grid">
            <div v-for="disk in data.disks" :key="disk.mountpoint" class="disk-card">
              <div class="disk-card-top">
                <a-progress
                  type="circle"
                  :percent="Number(disk.usage_percent.toFixed(2))"
                  :size="96"
                  :stroke-color="usageColor(disk.usage_percent)"
                  :stroke-width="8"
                />
                <div class="disk-card-info">
                  <div class="disk-mount" :title="disk.mountpoint">{{ disk.mountpoint }}</div>
                  <div class="disk-device" :title="disk.device">{{ disk.device }}</div>
                  <div class="disk-fs">{{ disk.fstype || '-' }}</div>
                </div>
              </div>
              <div class="disk-bar">
                <a-progress
                  type="line"
                  :percent="Number(disk.usage_percent.toFixed(2))"
                  :stroke-color="usageColor(disk.usage_percent)"
                  :show-info="false"
                  :stroke-width="8"
                />
              </div>
              <div class="disk-footer">
                <span>已用 {{ formatBytes(disk.used_bytes) }}</span>
                <span>总量 {{ formatBytes(disk.total_bytes) }}</span>
              </div>
            </div>
          </div>
          <div v-else class="empty-tip">暂无可用磁盘分区</div>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.home-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-bottom: 16px;
  box-sizing: border-box;
}

.home-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.welcome-card {
  background: var(--ad-color-bg-container);
  padding: 20px 24px;
  flex-shrink: 0;
}

.welcome-greet {
  font-size: 20px;
  font-weight: 600;
  color: var(--ad-color-text);
  line-height: 1.4;
}

.welcome-clock {
  margin-top: 8px;
  font-size: 14px;
  color: var(--ad-color-text-secondary);
  font-variant-numeric: tabular-nums;
}

.card-grid {
  display: grid;
  grid-template-columns: 1.25fr 1fr 1fr 1.1fr;
  grid-auto-rows: minmax(360px, 1fr);
  gap: 16px;
  align-items: stretch;
  min-height: calc(100vh - 220px);
}

.info-card {
  display: flex;
  flex-direction: column;
  min-height: 360px;
  height: 100%;
  background: var(--ad-color-bg-container);
  padding: 16px 20px 20px;
}

.card-body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.chart-body {
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.chart-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 180px;
}

.chart-meta {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.meta-name {
  font-size: 13px;
  color: var(--ad-color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: center;
}

.meta-row {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--ad-color-text-secondary);
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
  color: var(--ad-color-text);
}

.host-badge {
  display: inline-flex;
  align-self: flex-start;
  margin-bottom: 12px;
  padding: 4px 10px;
  background: var(--ad-color-primary-bg);
  color: var(--ad-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.kv {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  font-size: 13px;
  border-bottom: 1px solid var(--ad-color-border-secondary);
}

.kv span {
  color: var(--ad-color-text-secondary);
  flex-shrink: 0;
}

.kv b {
  font-weight: 500;
  color: var(--ad-color-text);
  text-align: right;
  word-break: break-all;
}

.host-foot {
  margin-top: auto;
  padding-top: 16px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.stat-block {
  background: var(--ad-color-bg-layout);
  padding: 10px 8px;
  text-align: center;
}

.stat-label {
  font-size: 12px;
  color: var(--ad-color-text-secondary);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 16px;
  font-weight: 600;
  color: var(--ad-color-text);
  word-break: break-all;
}

.stat-value.small {
  font-size: 12px;
}

.swap-hero {
  text-align: center;
  margin: 12px 0 20px;
}

.swap-percent {
  font-size: 42px;
  font-weight: 700;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

.swap-sub {
  margin-top: 8px;
  font-size: 13px;
  color: var(--ad-color-text-secondary);
}

.swap-legend {
  margin-top: auto;
  padding-top: 16px;
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--ad-color-text-secondary);
}

.disk-section {
  background: var(--ad-color-bg-container);
  padding: 16px 20px 20px;
  min-height: 320px;
}

.disk-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.disk-head .card-title {
  margin-bottom: 0;
}

.disk-count {
  font-size: 13px;
  color: var(--ad-color-text-secondary);
}

.disk-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

.disk-card {
  border: 1px solid var(--ad-color-border-secondary);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 220px;
}

.disk-card-top {
  display: flex;
  align-items: center;
  gap: 14px;
}

.disk-card-info {
  min-width: 0;
  flex: 1;
}

.disk-mount {
  font-size: 15px;
  font-weight: 600;
  color: var(--ad-color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disk-device {
  margin-top: 4px;
  font-size: 12px;
  color: var(--ad-color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disk-fs {
  margin-top: 6px;
  display: inline-block;
  padding: 2px 8px;
  font-size: 12px;
  background: var(--ad-color-bg-layout);
  color: var(--ad-color-text-secondary);
}

.disk-footer {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--ad-color-text-secondary);
}

.empty-tip {
  padding: 32px 0;
  text-align: center;
  color: var(--ad-color-text-secondary);
  font-size: 13px;
}

@media (max-width: 1400px) {
  .card-grid {
    grid-template-columns: 1.2fr 1fr;
    grid-auto-rows: minmax(320px, auto);
    min-height: auto;
  }
}

@media (max-width: 960px) {
  .card-grid {
    grid-template-columns: 1fr;
    grid-auto-rows: auto;
    min-height: auto;
  }

  .info-card {
    min-height: 280px;
  }
}
</style>
