import { apiGet } from '@/api/http'

export type HostOverview = {
  host: {
    hostname: string
    os: string
    platform: string
    platform_version: string
    kernel_version: string
    arch: string
    uptime_seconds: number
  }
  cpu: {
    model_name: string
    logical_cores: number
    physical_cores: number
    usage_percent: number
  }
  memory: {
    total_bytes: number
    used_bytes: number
    available_bytes: number
    usage_percent: number
  }
  swap: {
    total_bytes: number
    used_bytes: number
    available_bytes: number
    usage_percent: number
  }
  disks: Array<{
    device: string
    mountpoint: string
    fstype: string
    total_bytes: number
    used_bytes: number
    free_bytes: number
    usage_percent: number
  }>
  runtime: {
    num_cpu: number
  }
}

export function hostOverviewApi(signal?: AbortSignal) {
  return apiGet<HostOverview>('/system/host', signal ? { signal } : undefined)
}
