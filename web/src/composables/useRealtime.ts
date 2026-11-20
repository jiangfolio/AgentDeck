import { onUnmounted, ref, type Ref } from 'vue'
import {
  usePolling,
  type PollTask,
  type UsePollingOptions,
  type UsePollingReturn,
} from '@/composables/usePolling'

export type RealtimeFetcher<T> = (signal: AbortSignal) => Promise<T>

export type UseRealtimeOptions = UsePollingOptions & {
  /**
   * 后续轮询是否静默（不改 loading），避免整页闪烁；默认 true。
   * 首次无数据时仍会显示 loading。
   */
  silentRefresh?: boolean
  /** 连续失败提示节流（毫秒），默认 30000；传 0 表示每次都回调 */
  errorThrottleMs?: number
  onError?: (error: unknown) => void
  onSuccess?: () => void
}

export type UseRealtimeReturn<T> = UsePollingReturn & {
  data: Ref<T | null>
  loading: Ref<boolean>
  error: Ref<unknown>
  /** 最近一次成功拉取的时间戳（毫秒），未成功过为 0 */
  lastUpdated: Ref<number>
  refresh: () => Promise<void>
}

type RealtimeStore<T> = {
  data: Ref<T | null>
  loading: Ref<boolean>
  error: Ref<unknown>
  lastUpdated: Ref<number>
  pull: PollTask
  subscribers: number
}

/** 按 key 共享数据与 pull，配合 usePolling.key 做引用计数去重 */
const storeMap = new Map<string, RealtimeStore<unknown>>()

function isAbortError(e: unknown): boolean {
  if (!e || typeof e !== 'object') return false
  const err = e as { name?: string; code?: string }
  return err.name === 'CanceledError' || err.name === 'AbortError' || err.code === 'ERR_CANCELED'
}

/**
 * 构造拉取函数与状态；同一 key 复用同一份，保证只存在一路请求逻辑。
 */
function createStore<T>(
  fetcher: RealtimeFetcher<T>,
  options: UseRealtimeOptions,
): RealtimeStore<T> {
  const silentRefresh = options.silentRefresh !== false
  const errorThrottleMs = options.errorThrottleMs ?? 30_000

  const data = ref<T | null>(null) as Ref<T | null>
  const loading = ref(false)
  const error = ref<unknown>(null)
  const lastUpdated = ref(0)
  let lastErrorNotifyAt = 0

  const pull: PollTask = async (signal) => {
    // 1. 无缓存时展示 loading；静默刷新保留旧数据
    const showLoading = !silentRefresh || data.value === null
    if (showLoading) loading.value = true
    try {
      // 2. 带取消信号请求；隐藏/刷新 abort，避免叠请求打服务端
      const next = await fetcher(signal)
      if (signal.aborted) return
      data.value = next
      error.value = null
      lastUpdated.value = Date.now()
      options.onSuccess?.()
    } catch (e) {
      // 3. 主动取消不提示；其它错误节流回调
      if (isAbortError(e) || signal.aborted) return
      error.value = e
      const now = Date.now()
      if (
        options.onError &&
        (errorThrottleMs <= 0 || now - lastErrorNotifyAt >= errorThrottleMs)
      ) {
        lastErrorNotifyAt = now
        options.onError(e)
      }
    } finally {
      if (showLoading) loading.value = false
    }
  }

  return { data, loading, error, lastUpdated, pull, subscribers: 0 }
}

function getOrCreateStore<T>(
  key: string,
  fetcher: RealtimeFetcher<T>,
  options: UseRealtimeOptions,
): RealtimeStore<T> {
  const existed = storeMap.get(key) as RealtimeStore<T> | undefined
  if (existed) return existed
  const created = createStore(fetcher, options)
  storeMap.set(key, created as RealtimeStore<unknown>)
  return created
}

/**
 * 可复用的实时数据拉取：隐藏停轮询并取消在途、刷新清理、可选 key 全局去重。
 */
export function useRealtime<T>(
  fetcher: RealtimeFetcher<T>,
  options: UseRealtimeOptions = {},
): UseRealtimeReturn<T> {
  const shareKey = options.key?.trim()

  // 1. 有 key 则复用 store；无 key 则组件私有一份
  const store = shareKey
    ? getOrCreateStore(shareKey, fetcher, options)
    : createStore(fetcher, options)

  if (shareKey) {
    store.subscribers += 1
  }

  // 2. 每次订阅都挂到 usePolling（同 key 共用一个定时器，按引用计数启停）
  const polling = usePolling(store.pull, options)

  if (shareKey) {
    onUnmounted(() => {
      // 3. 最后一位订阅者离开时丢掉缓存，避免热更新/重进残留脏订阅计数
      store.subscribers -= 1
      if (store.subscribers <= 0) {
        storeMap.delete(shareKey)
      }
    })
  }

  return {
    ...polling,
    data: store.data,
    loading: store.loading,
    error: store.error,
    lastUpdated: store.lastUpdated,
    refresh: polling.trigger,
  }
}
