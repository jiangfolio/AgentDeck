import {
  nextTick,
  onActivated,
  onBeforeUnmount,
  onMounted,
  ref,
  type Ref,
} from 'vue'

export type UseTableScrollYOptions = {
  /** 页面根；不传则用 wrapRef / .ad-page-fill */
  rootRef?: Ref<HTMLElement | null | undefined>
  /** 额外预留（汇总区等）；rect 从可见表头起算时一般不用再扣工具栏 */
  extraOffset?: number | (() => number)
  /** resize 节流 ms */
  throttleMs?: number
}

/**
 * 计算 a-table 的 scroll.y（仅作表体 max-height）。
 * 对齐 AccountBook：外置分页贴底，用 getBoundingClientRect 实测
 * 「可见表头底 → 内容区底 − 分页」。
 */
export function useTableScrollY(options: UseTableScrollYOptions = {}) {
  const scrollY = ref(300)
  /** 绑在 .ad-page-fill 上，作为内容区测量盒 */
  const wrapRef = ref<HTMLElement | null>(null) as Ref<HTMLElement | null>
  const throttleMs = options.throttleMs ?? 200
  const SAFETY_GAP = 16

  let resizeTimer: number | undefined
  let observer: ResizeObserver | undefined

  function extraOf(): number {
    return typeof options.extraOffset === 'function'
      ? options.extraOffset()
      : options.extraOffset || 0
  }

  function contentBox(): HTMLElement | null {
    return (
      options.rootRef?.value ||
      wrapRef.value ||
      (document.querySelector('.ad-page-fill') as HTMLElement | null)
    )
  }

  function scopeRoot(): ParentNode {
    return contentBox() || document
  }

  /** 当前可见表头（跳过隐藏 pane） */
  function findVisibleThead(root: ParentNode): HTMLElement | null {
    const nodes = root.querySelectorAll?.('.ant-table-thead')
    if (!nodes) return null
    for (const el of Array.from(nodes)) {
      if (!(el instanceof HTMLElement)) continue
      const rect = el.getBoundingClientRect()
      if (rect.height > 0 && rect.width > 0) return el
    }
    return null
  }

  function findPagination(root: ParentNode): HTMLElement | null {
    return (
      (root.querySelector?.('.ad-page-pagination') as HTMLElement | null) ||
      (root.querySelector?.('.ant-pagination') as HTMLElement | null)
    )
  }

  /**
   * 1. 优先：表头底 → 内容区底 − 外置分页 − 安全间隙
   * 2. 兜底：容器高度 − 表头 − 分页 − 额外预留
   */
  function measureByRect(): number | null {
    const box = contentBox()
    if (!box) return null
    const root = scopeRoot()
    const thead = findVisibleThead(root)
    if (!thead) return null

    const page = findPagination(root)
    const pageH = page?.offsetHeight || 0
    const boxBottom = box.getBoundingClientRect().bottom
    const topEdge = thead.getBoundingClientRect().bottom
    const next = Math.floor(boxBottom - topEdge - pageH - SAFETY_GAP - extraOf())
    return next > 120 ? next : 120
  }

  function measureByOffsetFallback(): number {
    const box = contentBox()
    if (!box) return 300
    const root = scopeRoot()
    const thead = findVisibleThead(root)
    const page = findPagination(root)
    const due =
      (thead?.offsetHeight || 39) +
      (page?.offsetHeight || 0) +
      SAFETY_GAP +
      extraOf()
    const next = Math.floor(box.clientHeight - due)
    return next > 120 ? next : 120
  }

  /** 内容区纵向溢出时回缩表体上限 */
  function correctOverflow() {
    const box = contentBox()
    if (!box) return
    const overflow = box.scrollHeight - box.clientHeight
    if (overflow <= 0) return
    scrollY.value = Math.max(120, scrollY.value - overflow - 2)
  }

  /** 空余时回填表体，避免分页悬空过高 */
  function correctUnused() {
    const box = contentBox()
    if (!box) return
    const unused = box.clientHeight - box.scrollHeight
    if (unused <= 8) return
    scrollY.value = scrollY.value + unused - 4
  }

  function settle() {
    correctOverflow()
    correctUnused()
    correctOverflow()
  }

  function recalc() {
    nextTick(() => {
      window.setTimeout(() => {
        scrollY.value = measureByRect() ?? measureByOffsetFallback()
        requestAnimationFrame(() => {
          settle()
          requestAnimationFrame(settle)
        })
      }, 50)
    })
  }

  function onResize() {
    if (resizeTimer) window.clearTimeout(resizeTimer)
    resizeTimer = window.setTimeout(() => recalc(), throttleMs)
  }

  onMounted(() => {
    recalc()
    window.setTimeout(recalc, 200)
    window.setTimeout(recalc, 500)
    window.addEventListener('resize', onResize)
    observer = new ResizeObserver(() => recalc())
    if (wrapRef.value) observer.observe(wrapRef.value)
  })

  onActivated(() => {
    recalc()
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', onResize)
    if (resizeTimer) window.clearTimeout(resizeTimer)
    observer?.disconnect()
  })

  return { wrapRef, scrollY, recalc }
}
