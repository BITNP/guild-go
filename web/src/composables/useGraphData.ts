import { onBeforeUnmount, ref, shallowRef } from 'vue'
import { fetchGraph, type GraphNode } from '@/lib/api'
import { errorMessage } from '@/lib/utils'

const PAGE_SIZE = 200

export function useGraphData() {
  const nodes = shallowRef<GraphNode[]>([])
  const loading = ref(false)
  const complete = ref(false)
  const error = ref<string | null>(null)
  const loaded = ref(0)
  let generation = 0
  let disposed = false

  async function load() {
    const currentGeneration = ++generation
    const merged = new Map<number, GraphNode>()
    nodes.value = []
    loading.value = true
    complete.value = false
    error.value = null
    loaded.value = 0

    try {
      let cursor: number | null = null
      do {
        const page = await fetchGraph(cursor, PAGE_SIZE)
        if (disposed || currentGeneration !== generation) return
        for (const node of page.items) merged.set(node.id, node)
        nodes.value = [...merged.values()]
        loaded.value = merged.size
        cursor = page.nextCursor
        complete.value = !page.hasMore
        if (page.hasMore) await wait(140)
      } while (cursor != null && !disposed && currentGeneration === generation)
    } catch (err) {
      if (currentGeneration === generation) {
        error.value = errorMessage(err, '无法加载关系图谱')
      }
    } finally {
      if (currentGeneration === generation) loading.value = false
    }
  }

  onBeforeUnmount(() => {
    disposed = true
    generation++
  })

  return { nodes, loading, complete, error, loaded, load }
}

function wait(milliseconds: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds))
}
