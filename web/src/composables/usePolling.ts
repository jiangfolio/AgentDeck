import { onMounted, onUnmounted, ref, type Ref } from 'vue'

export type PollTask = (signal: AbortSignal) => void | Promise<void>

export type UsePollingOptions = {
  /** 轮询间隔（毫秒），默认 5000 */
  interval?: number
  /** 挂载后是否立即执行一次，默认 true */
  immediate?: boolean
  /** 标签页隐藏时是否暂停并中止在途请求，默认 true */
  pauseOnHidden?: boolean
  /** 是否在挂载时自动启动，默认 true */
  autoStart?: boolean
  /**
   * 全局去重键：同 key 在整页生命周期内只保留一份定时器。
   * 用于路由反复进入、组件重挂载时避免叠多路轮询打爆服务端。
   */
  key?: string
}

export type UsePollingReturn = {
  /** 当前是否处于轮询中（含因隐藏而暂停时仍为 true） */
  active: Ref<boolean>
  /** 是否因页面不可见而暂时挂起 */
  paused: Ref<boolean>
  start: () => void
  stop: () => void
  /** 立即触发一次任务（不打断定时节奏；隐藏中不会发请求） */
  trigger: () => Promise<void>
}

type SharedEntry = {
  refCount: number
  api: UsePollingReturn
  stopOwned: () => void
}

/** 按 key 共享的轮询实例，防止同页多处订阅重复打点 */
const sharedPolls = new Map<string, SharedEntry>()

function isAbortError(e: unknown): boolean {
  if (!e || typeof e !== 'object') return false
  const err = e as { name?: string; code?: string }
  return err.name === 'CanceledError' || err.name === 'AbortError' || err.code === 'ERR_CANCELED'
}

/**
 * 创建单实例轮询控制器（不含 Vue 生命周期绑定）。
 */
function createPollingController(
  task: PollTask,
  options: UsePollingOptions,
): UsePollingReturn & { dispose: () => void } {
  const interval = Math.max(200, options.interval ?? 5000)
  const immediate = options.immediate !== false
  const pauseOnHidden = options.pauseOnHidden !== false

  const active = ref(false)
  const paused = ref(false)

  let timer: number | undefined
  let running = false
  let disposed = false
  let abortCurrent: AbortController | undefined
  /** 暂停/中止期间若有人触发，可见后补跑一次，避免漏更新 */
  let runAfterResume = false

  function clearTimer() {
    if (timer !== undefined) {
      window.clearInterval(timer)
      timer = undefined
    }
  }

  function abortInFlight() {
    if (!abortCurrent) return
    abortCurrent.abort()
    abortCurrent = undefined
  }

  /**
   * 执行一次任务；隐藏/卸载中不发新请求，上一轮未完成则跳过。
   */
  async function runOnce() {
    // 1. 已销毁、已暂停、标签隐藏：禁止新请求，从源头挡住重复轮询
    if (disposed || paused.value) return
    if (pauseOnHidden && document.visibilityState === 'hidden') return
    // 2. 上一轮还在飞：跳过本拍，避免同实例并发叠压
    if (running) return

    // 3. 带 AbortSignal 执行；隐藏/卸载会 abort，axios 随即取消
    running = true
    const ac = new AbortController()
    abortCurrent = ac
    try {
      await task(ac.signal)
    } catch (e) {
      if (!isAbortError(e)) {
        // 业务错误由 task 内部消化；此处仅吞掉 abort，避免未处理 rejection
      }
    } finally {
      if (abortCurrent === ac) abortCurrent = undefined
      running = false
      // 4. 若暂停期间有人要求补跑，恢复后立刻再拉一次
      if (runAfterResume && !disposed && !paused.value) {
        runAfterResume = false
        void runOnce()
      }
    }
  }

  function schedule() {
    clearTimer()
    timer = window.setInterval(() => {
      void runOnce()
    }, interval)
  }

  /**
   * 按可见性挂起或恢复；隐藏时清定时器并中止在途请求。
   */
  function syncVisibility() {
    if (!active.value || !pauseOnHidden || disposed) return
    // 1. 隐藏：停表 + 取消在途，杜绝后台持续打服务端
    if (document.visibilityState === 'hidden') {
      paused.value = true
      clearTimer()
      abortInFlight()
      return
    }
    // 2. 重新可见：恢复节奏；若上一轮 abort 尚未结束则标记补跑
    paused.value = false
    schedule()
    if (running) {
      runAfterResume = true
      return
    }
    void runOnce()
  }

  function onVisibilityChange() {
    syncVisibility()
  }

  /**
   * 刷新/关页时尽早停表并 abort，不依赖 Vue 卸载时机。
   */
  function onPageHide() {
    if (!active.value || disposed) return
    clearTimer()
    abortInFlight()
  }

  function start() {
    if (disposed || active.value) return
    active.value = true

    if (pauseOnHidden) {
      document.addEventListener('visibilitychange', onVisibilityChange)
    }
    window.addEventListener('pagehide', onPageHide)

    // 1. 若启动时已隐藏：只挂监听，不发请求、不开定时器
    if (pauseOnHidden && document.visibilityState === 'hidden') {
      paused.value = true
      return
    }

    // 2. 可见时：按需立即拉一次，再进入固定间隔
    paused.value = false
    if (immediate) {
      void runOnce()
    }
    schedule()
  }

  function stop() {
    if (!active.value && !timer && !abortCurrent) {
      active.value = false
      paused.value = false
      return
    }
    active.value = false
    paused.value = false
    runAfterResume = false
    clearTimer()
    abortInFlight()
    if (pauseOnHidden) {
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
    window.removeEventListener('pagehide', onPageHide)
  }

  async function trigger() {
    if (disposed || paused.value) return
    if (pauseOnHidden && document.visibilityState === 'hidden') return
    if (running) {
      runAfterResume = true
      return
    }
    await runOnce()
  }

  function dispose() {
    disposed = true
    stop()
  }

  return { active, paused, start, stop, trigger, dispose }
}

/**
 * 通用轮询：定时执行任务，隐藏暂停/中止在途、刷新 pagehide 清理、可选 key 全局去重。
 */
export function usePolling(task: PollTask, options: UsePollingOptions = {}): UsePollingReturn {
  const shareKey = options.key?.trim()

  // 无 key：组件私有一份，挂载启、卸载停
  if (!shareKey) {
    const ctl = createPollingController(task, options)
    onMounted(() => {
      if (options.autoStart !== false) ctl.start()
    })
    onUnmounted(() => {
      ctl.dispose()
    })
    return {
      active: ctl.active,
      paused: ctl.paused,
      start: ctl.start,
      stop: ctl.stop,
      trigger: ctl.trigger,
    }
  }

  // 有 key：整页共享同一控制器，refCount 归零才真正销毁
  let entry = sharedPolls.get(shareKey)
  if (!entry) {
    const ctl = createPollingController(task, options)
    entry = {
      refCount: 0,
      api: {
        active: ctl.active,
        paused: ctl.paused,
        start: ctl.start,
        stop: () => {
          /* 共享实例不允许单订阅方直接 stop，由 refCount 管理 */
        },
        trigger: ctl.trigger,
      },
      stopOwned: ctl.dispose,
    }
    sharedPolls.set(shareKey, entry)
    if (options.autoStart !== false) {
      ctl.start()
    }
  }

  entry.refCount += 1
  const current = entry

  onUnmounted(() => {
    current.refCount -= 1
    if (current.refCount > 0) return
    current.stopOwned()
    sharedPolls.delete(shareKey)
  })

  return current.api
}
