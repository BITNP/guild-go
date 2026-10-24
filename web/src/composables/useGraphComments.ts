import { ref } from 'vue'
import { fetchComments, type Comment } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

export function useGraphComments() {
  const items = ref<Comment[]>([])
  const loading = ref(false)
  const loadingOlder = ref(false)
  const hasMore = ref(false)
  const error = ref<string | null>(null)
  let nextBefore: number | null = null
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
      const page = await fetchComments(id)
      if (current !== generation) return
      items.value = [...page.items].reverse()
      hasMore.value = page.hasMore
      nextBefore = page.nextBefore
    } catch (err) {
      if (current === generation) error.value = errorMessage(err, '无法加载节点评注')
    } finally {
      if (current === generation) loading.value = false
    }
  }

  async function loadOlder() {
    if (nodeId == null || loadingOlder.value || !hasMore.value || nextBefore == null) return
    const current = generation
    loadingOlder.value = true
    error.value = null
    try {
      const page = await fetchComments(nodeId, nextBefore)
      if (current !== generation) return
      items.value = [...page.items].reverse().concat(items.value)
      hasMore.value = page.hasMore
      nextBefore = page.nextBefore
    } catch (err) {
      if (current === generation) error.value = errorMessage(err, '无法加载更早的评注')
    } finally {
      if (current === generation) loadingOlder.value = false
    }
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

  return { items, loading, loadingOlder, hasMore, error, load, loadOlder, clear }
}
