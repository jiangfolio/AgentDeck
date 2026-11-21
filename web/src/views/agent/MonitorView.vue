<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { message } from 'antdv-next'
import { monitorSnapshotApi, type MonitorSnapshot } from '@/api/agent'
import { usePolling } from '@/composables/usePolling'

type HostPoint = {
  label: string
  cpu: number
  mem: number
  load1: number
}

/** 下方主图序列：流量时为上下行，磁盘时为读写。 */
type IoPoint = {
  label: string
  a: number
  b: number
}

const MAX = 60
const snap = ref<MonitorSnapshot | null>(null)
const hostPoints = ref<HostPoint[]>([])
const ioPoints = ref<IoPoint[]>([])
const section = ref<'net' | 'disk'>('net')
const nic = ref('all')
const diskName = ref('all')

const cpuEl = ref<HTMLElement | null>(null)
const memEl = ref<HTMLElement | null>(null)
const loadEl = ref<HTMLElement | null>(null)
const mainEl = ref<HTMLElement | null>(null)
let cpuChart: echarts.ECharts | null = null
let memChart: echarts.ECharts | null = null
let loadChart: echarts.ECharts | null = null
let mainChart: echarts.ECharts | null = null

function formatRate(n: number) {
  if (!n) return '0 B/s'
  const u = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(v >= 10 || i === 0 ? 0 : 2)} ${u[i]}`
}

function formatBytes(n: number) {
  if (!n) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(2)} ${u[i]}`
}

function clock(ts: number) {
  const d = new Date(ts)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

function pickRates(s: MonitorSnapshot) {
  let netUp = 0
  let netDown = 0
  if (nic.value === 'all') {
    for (const it of s.net.interfaces || []) {
      netUp += it.bytes_sent_per_sec || 0
      netDown += it.bytes_recv_per_sec || 0
    }
  } else {
    const it = (s.net.interfaces || []).find((x) => x.name === nic.value)
    netUp = it?.bytes_sent_per_sec || 0
    netDown = it?.bytes_recv_per_sec || 0
  }
  let diskR = 0
  let diskW = 0
  if (diskName.value === 'all') {
    diskR = s.disk_io.read_bytes_per_sec || 0
    diskW = s.disk_io.write_bytes_per_sec || 0
  } else {
    const it = (s.disk_io.devices || []).find((x) => x.name === diskName.value)
    diskR = it?.read_bytes_per_sec || 0
    diskW = it?.write_bytes_per_sec || 0
  }
  return { netUp, netDown, diskR, diskW }
}

function ioSample(s: MonitorSnapshot): IoPoint {
  const rate = pickRates(s)
  return {
    label: clock(s.ts),
    a: section.value === 'net' ? rate.netUp : rate.diskR,
    b: section.value === 'net' ? rate.netDown : rate.diskW,
  }
}

function pushPoint(s: MonitorSnapshot) {
  // 1. 顶部负载/CPU/内存始终累加，切换流量与磁盘不影响
  hostPoints.value = [
    ...hostPoints.value,
    {
      label: clock(s.ts),
      cpu: s.cpu.usage_percent || 0,
      mem: s.memory.usage_percent || 0,
      load1: s.load.load1 || 0,
    },
  ].slice(-MAX)
  // 2. 主图按当前分区单独累加
  ioPoints.value = [...ioPoints.value, ioSample(s)].slice(-MAX)
}

function lineOption(name: string, labels: string[], data: number[], color: string) {
  return {
    animation: false,
    grid: { left: 8, right: 8, top: 8, bottom: 8 },
    xAxis: { type: 'category', show: false, data: labels },
    yAxis: { type: 'value', show: false },
    series: [{ name, type: 'line', showSymbol: false, data, lineStyle: { color, width: 1.5 }, areaStyle: { color, opacity: 0.15 } }],
  }
}

function renderHost() {
  const labels = hostPoints.value.map((p) => p.label)
  cpuChart?.setOption(lineOption('CPU', labels, hostPoints.value.map((p) => p.cpu), '#1677ff'), true)
  memChart?.setOption(lineOption('内存', labels, hostPoints.value.map((p) => p.mem), '#52c41a'), true)
  loadChart?.setOption(lineOption('负载', labels, hostPoints.value.map((p) => p.load1), '#faad14'), true)
}

function renderMain() {
  const labels = ioPoints.value.map((p) => p.label)
  const up = ioPoints.value.map((p) => p.a)
  const down = ioPoints.value.map((p) => p.b)
  mainChart?.setOption(
    {
      animation: false,
      tooltip: { trigger: 'axis' },
      legend: { data: section.value === 'net' ? ['上行', '下行'] : ['读', '写'], right: 12 },
      grid: { left: 48, right: 16, top: 32, bottom: 28 },
      xAxis: { type: 'category', data: labels, boundaryGap: false },
      yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatRate(v).replace('/s', '') } },
      series: [
        { name: section.value === 'net' ? '上行' : '读', type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.25 }, data: up },
        { name: section.value === 'net' ? '下行' : '写', type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.15 }, data: down },
      ],
    },
    true,
  )
}

function render() {
  renderHost()
  renderMain()
}

/** 只重置主图历史；顶部曲线保持连续。 */
function resetIoSeries() {
  ioPoints.value = snap.value ? [ioSample(snap.value)] : []
  renderMain()
}

async function tick(signal?: AbortSignal) {
  try {
    const data = await monitorSnapshotApi(signal)
    snap.value = data
    if (nic.value !== 'all' && !(data.net.interfaces || []).some((x) => x.name === nic.value)) nic.value = 'all'
    if (diskName.value !== 'all' && !(data.disk_io.devices || []).some((x) => x.name === diskName.value)) diskName.value = 'all'
    pushPoint(data)
    render()
  } catch (e: any) {
    if (e?.code === 'ERR_CANCELED' || e?.name === 'CanceledError') return
    message.error(e?.message || '采集监控失败')
  }
}

const currentNet = computed(() => {
  if (!snap.value) return { up: 0, down: 0, sent: 0, recv: 0 }
  const rate = pickRates(snap.value)
  let sent = snap.value.net.total_sent_bytes
  let recv = snap.value.net.total_recv_bytes
  if (nic.value !== 'all') {
    const it = snap.value.net.interfaces.find((x) => x.name === nic.value)
    sent = it?.bytes_sent || 0
    recv = it?.bytes_recv || 0
  }
  return { up: rate.netUp, down: rate.netDown, sent, recv }
})

usePolling((signal) => tick(signal), { interval: 2000, immediate: true, key: 'agent-monitor' })

function onResize() {
  cpuChart?.resize()
  memChart?.resize()
  loadChart?.resize()
  mainChart?.resize()
}

onMounted(() => {
  if (cpuEl.value) cpuChart = echarts.init(cpuEl.value)
  if (memEl.value) memChart = echarts.init(memEl.value)
  if (loadEl.value) loadChart = echarts.init(loadEl.value)
  if (mainEl.value) mainChart = echarts.init(mainEl.value)
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  cpuChart?.dispose()
  memChart?.dispose()
  loadChart?.dispose()
  mainChart?.dispose()
})

watch(section, resetIoSeries)
watch([nic, diskName], resetIoSeries)
</script>

<template>
  <div class="ad-page-scroll monitor-page">
    <div class="cards">
      <div class="card">
        <div class="card-title">平均负载</div>
        <div v-if="snap && !snap.load.supported" class="unsupported">当前平台不支持</div>
        <template v-else>
          <div class="nums">
            <span>1 分钟 {{ snap?.load.load1?.toFixed(2) || '0.00' }}</span>
            <span>5 分钟 {{ snap?.load.load5?.toFixed(2) || '0.00' }}</span>
            <span>15 分钟 {{ snap?.load.load15?.toFixed(2) || '0.00' }}</span>
          </div>
          <div ref="loadEl" class="spark" />
        </template>
      </div>
      <div class="card">
        <div class="card-title">CPU {{ snap?.cpu.usage_percent?.toFixed(1) || '0.0' }}%</div>
        <div ref="cpuEl" class="spark" />
      </div>
      <div class="card">
        <div class="card-title">
          内存 {{ snap?.memory.usage_percent?.toFixed(1) || '0.0' }}%
        </div>
        <div class="sub">
          {{ formatBytes(snap?.memory.used_bytes || 0) }} / {{ formatBytes(snap?.memory.total_bytes || 0) }}
        </div>
        <div ref="memEl" class="spark" />
      </div>
    </div>

    <div class="card main-card">
      <div class="main-head">
        <a-radio-group v-model:value="section" button-style="solid">
          <a-radio-button value="net">流量</a-radio-button>
          <a-radio-button value="disk">磁盘 IO</a-radio-button>
        </a-radio-group>
        <a-select v-if="section === 'net'" v-model:value="nic" style="width: 180px" :options="[{ value: 'all', label: '网卡 所有' }, ...(snap?.net.interfaces || []).map((x) => ({ value: x.name, label: x.name }))]" />
        <a-select v-else v-model:value="diskName" style="width: 180px" :options="[{ value: 'all', label: '磁盘 所有' }, ...(snap?.disk_io.devices || []).map((x) => ({ value: x.name, label: x.name }))]" />
      </div>
      <div class="tags">
        <template v-if="section === 'net'">
          <a-tag color="blue">上行: {{ formatRate(currentNet.up) }}</a-tag>
          <a-tag color="blue">下行: {{ formatRate(currentNet.down) }}</a-tag>
          <a-tag>总发送: {{ formatBytes(currentNet.sent) }}</a-tag>
          <a-tag>总接收: {{ formatBytes(currentNet.recv) }}</a-tag>
        </template>
        <template v-else>
          <a-tag color="blue">读: {{ formatRate(ioPoints.at(-1)?.a || 0) }}</a-tag>
          <a-tag color="gold">写: {{ formatRate(ioPoints.at(-1)?.b || 0) }}</a-tag>
        </template>
      </div>
      <div class="main-body">
        <div ref="mainEl" class="main-chart" />
        <a-table
          class="top-table"
          size="small"
          :pagination="false"
          row-key="pid"
          :data-source="snap?.top || []"
          :columns="[
            { title: 'PID', dataIndex: 'pid', key: 'pid', width: 80 },
            { title: '用户', dataIndex: 'user', key: 'user', width: 80, ellipsis: true },
            { title: '进程', dataIndex: 'name', key: 'name', ellipsis: true },
            { title: '占比', dataIndex: 'cpu_percent', key: 'cpu', width: 70 },
          ]"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.monitor-page {
  background: var(--ad-color-bg-layout);
  padding: 12px;
}

.cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.card {
  background: var(--ad-color-bg-container);
  padding: 12px 16px;
}

.card-title {
  font-weight: 600;
  margin-bottom: 8px;
}

.nums,
.sub {
  color: var(--ad-color-text-secondary);
  font-size: 12px;
  display: flex;
  gap: 12px;
}

.spark {
  height: 72px;
}

.unsupported {
  color: var(--ad-color-text-secondary);
  padding: 24px 0;
}

.main-head,
.tags {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.main-body {
  display: grid;
  grid-template-columns: 1fr 360px;
  gap: 12px;
}

.main-chart {
  height: 320px;
}

@media (max-width: 1100px) {
  .cards,
  .main-body {
    grid-template-columns: 1fr;
  }
}
</style>
