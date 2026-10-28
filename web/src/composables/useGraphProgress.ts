import { ref } from 'vue'
import {
  fetchProgress,
  type Comment,
  type ProgressCursor,
  type ProgressItem,
  type Reaction,
} from '@/lib/api'
import { errorMessage } from '@/lib/utils'

function progressKey(item: ProgressItem): string {
  return item.kind === 'comment'
    ? `comment:${item.comment.id}`
    : `description_change:${item.descriptionChange.id}`
}

export function useGraphProgress() {
  const items = ref<ProgressItem[]>([])
  const loading = ref(false)
  const loadingOlder = ref(false)
  const hasMore = ref(false)
  const error = ref<string | null>(null)
  let nextBefore: ProgressCursor | null = null
  let generation = 0
  let nodeId: number | null = null

  async function load(id: number) {
    const current = ++generation
    nodeId = id
    items.value = []
    loading.value = true
    loadingOlder.value = false
    hasMore.value = false
    error.value = null
    nextBefore = null
    try {
      const page = await fetchProgress(id)
      if (current !== generation) return
      const fetched = [...page.items].reverse()
      const fetchedKeys = new Set(fetched.map(progressKey))
      const locallyAdded = items.value.filter(
        (item) => !fetchedKeys.has(progressKey(item)),
      )
      items.value = [...fetched, ...locallyAdded]
      hasMore.value = page.hasMore
      nextBefore = page.nextBefore
    } catch (err) {
      if (current === generation) error.value = errorMessage(err, '无法加载节点进展')
    } finally {
      if (current === generation) loading.value = false
    }
  }

  async function loadOlder() {
    if (
      nodeId == null ||
      loadingOlder.value ||
      !hasMore.value ||
      nextBefore == null
    ) {
      return
    }
    const current = generation
    loadingOlder.value = true
    error.value = null
    try {
      const page = await fetchProgress(nodeId, nextBefore)
      if (current !== generation) return
      items.value = [...page.items].reverse().concat(items.value)
      hasMore.value = page.hasMore
      nextBefore = page.nextBefore
    } catch (err) {
      if (current === generation) {
        error.value = errorMessage(err, '无法加载更早的进展')
      }
    } finally {
      if (current === generation) loadingOlder.value = false
    }
  }

  function add(comment: Comment) {
    if (nodeId !== comment.nodeId) return
    const key = `comment:${comment.id}`
    if (items.value.some((item) => progressKey(item) === key)) return
    items.value = [...items.value, { kind: 'comment', comment }]
  }

  function updateReactions(commentId: number, reactions: Reaction[]) {
    items.value = items.value.map((item) =>
      item.kind === 'comment' && item.comment.id === commentId
        ? { ...item, comment: { ...item.comment, reactions } }
        : item,
    )
  }

  function clear() {
    generation++
    nodeId = null
    items.value = []
    loading.value = false
    loadingOlder.value = false
    hasMore.value = false
    nextBefore = null
    error.value = null
  }

  return {
    items,
    loading,
    loadingOlder,
    hasMore,
    error,
    load,
    loadOlder,
    add,
    updateReactions,
    clear,
  }
}
