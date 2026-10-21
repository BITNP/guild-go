import { computed, ref } from 'vue'
import { fetchNodes, type NodeSummary } from '@/lib/api'

const nodes = ref<NodeSummary[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
let inflight: Promise<void> | null = null

const byId = computed(() => {
  const map = new Map<number, NodeSummary>()
  for (const node of nodes.value) map.set(node.id, node)
  return map
})

const roots = computed(() =>
  nodes.value
    .filter((node) => node.parentId == null)
    .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id),
)

function childrenOf(parentId: number): NodeSummary[] {
  return nodes.value
    .filter((node) => node.parentId === parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
}

function ancestorsOf(nodeId: number): NodeSummary[] {
  const result: NodeSummary[] = []
  let current = byId.value.get(nodeId)
  const guard = new Set<number>()
  while (current && !guard.has(current.id)) {
    guard.add(current.id)
    result.unshift(current)
    current = current.parentId == null ? undefined : byId.value.get(current.parentId)
  }
  return result
}

async function load(force = false): Promise<void> {
  if (inflight && !force) return inflight
  inflight = (async () => {
    loading.value = true
    try {
      nodes.value = await fetchNodes()
      error.value = null
    } catch (err) {
      error.value = err instanceof Error ? err.message : '无法加载节点树'
    } finally {
      loading.value = false
      inflight = null
    }
  })()
  return inflight
}

async function reload(): Promise<void> {
  await load(true)
}

export function useNodeStore() {
  return {
    nodes,
    byId,
    roots,
    loading,
    error,
    load,
    reload,
    childrenOf,
    ancestorsOf,
  }
}